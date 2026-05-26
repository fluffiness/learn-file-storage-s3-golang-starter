package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("Uploading thumbnail for video", videoID, "by user", userID)

	// Setup memory limit. If the file is over the limit, a tmp file will be created.
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	// Getting thumbnail data and metadata
	tnFile, tnHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Cannot read thumbnail file", err)
		return
	}
	defer tnFile.Close()
	mediaTypeHeader := tnHeader.Header.Get("Content-Type")
	mimeType, _, err := mime.ParseMediaType(mediaTypeHeader)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to parse mime type", err)
		return
	}
	if mimeType != "image/jpeg" && mimeType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Thumbnail must be of mime type image/jpeg or image/png", nil)
		return
	}

	// Store thumbnail in the file system
	ext := strings.Split(mimeType, "/")[len(strings.Split(mimeType, "/"))-1]
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	thumbnailFileName := base64.RawURLEncoding.EncodeToString(randBytes) + "." + ext
	thumbnailFilePath := filepath.Join(cfg.assetsRoot, thumbnailFileName)
	fmt.Println(thumbnailFilePath)
	thumbnailFile, err := os.Create(thumbnailFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create thumbnail file in server file system", err)
		return
	}
	_, err = io.Copy(thumbnailFile, tnFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to write thumbnail data to server assets file", err)
		return
	}

	// Update video metadata in the database with thumbnailURL
	thumbnailURL := "http://localhost:" + cfg.port + "/assets/" + thumbnailFileName
	metaData.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(metaData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video metadata with thumbnail URL", err)
		return
	}

	// Respond with updated metadata
	respondWithJSON(w, http.StatusOK, metaData)
}
