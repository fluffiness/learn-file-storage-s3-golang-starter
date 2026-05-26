package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// Check if the video ID exists in the database
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	// validate access token (JWT)
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	// Getting associated video metadata
	metaData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Video doesn't exist. Failed to fetch metadata from database", err)
		return
	}
	if metaData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not the author of the video", nil)
		return
	}

	fmt.Println("Uploading video", videoID, "by user", userID)

	// Setup memory limit. If the file is over the limit, a tmp file will be created.
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	// Getting video data and metadata
	vFile, vHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Cannot read video file", err)
		return
	}
	defer vFile.Close()
	mediaTypeHeader := vHeader.Header.Get("Content-Type")
	mimeType, _, err := mime.ParseMediaType(mediaTypeHeader)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to parse mime type", err)
		return
	}
	if mimeType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Thumbnail must be of mime type image/jpeg or image/png", nil)
		return
	}

	// Create tmp file to store video data
	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create tmp file", err)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	_, err = io.Copy(tmpFile, vFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create tmp file", err)
		return
	}
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to reset read offset of tmp file", err)
	}

	// Process video to move the moov atom to the start for quick start
	processedVideoPath, err := processVideoForFastStart(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to move the moov atom to the start of the video", err)
		return
	}
	processedVideoFile, err := os.Open(processedVideoPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to read processed video file", err)
		return
	}

	// Determine aspect ratio of the video
	aspectRatio, err := getVideoAspectRatio(processedVideoFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get video aspect ratio", err)
	}
	var s3KeyPrefix string
	switch aspectRatio {
	case "16:9":
		s3KeyPrefix = "landscape/"
	case "9:16":
		s3KeyPrefix = "portrait/"
	default:
		s3KeyPrefix = "other/"
	}

	// Store file in AWS S3
	ext := strings.Split(mimeType, "/")[len(strings.Split(mimeType, "/"))-1]
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	s3Key := s3KeyPrefix + base64.RawURLEncoding.EncodeToString(randBytes) + "." + ext
	s3PutParams := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &s3Key,
		Body:        processedVideoFile,
		ContentType: &mimeType,
	}
	_, err = cfg.s3Client.PutObject(r.Context(), &s3PutParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to store video in AWS S3", err)
		return
	}

	// Update video metadata with S3 URL
	videoURL := fmt.Sprintf("https://%s/%s", cfg.s3CfDistribution, s3Key)
	metaData.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(metaData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video URL in database", err)
		return
	}

	fmt.Printf("Uploaded video URL: %s\n", videoURL)

	// Respond with updated metadata
	// metaData, err = cfg.dbVideoToSignedVideo(metaData)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Failed to generate presigned video URL", err)
	// 	return
	// }
	respondWithJSON(w, http.StatusOK, metaData)
}
