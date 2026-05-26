package main

import (
	"bytes"
	"encoding/json"
	"math"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	inspectVideo := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	buffer := bytes.NewBuffer([]byte{})
	inspectVideo.Stdout = buffer
	err := inspectVideo.Run()
	if err != nil {
		return "", err
	}

	type stream struct {
		Index     int    `json:"index"`
		CodecType string `json:"codec_type"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}
	type videoMetadata struct {
		Streams []stream `json:"streams"`
	}
	var metadata videoMetadata
	err = json.Unmarshal(buffer.Bytes(), &metadata)
	if err != nil {
		return "", err
	}

	aspectRatio := float64(metadata.Streams[0].Width) / float64(metadata.Streams[0].Height)
	tolerance := 0.01
	if math.Abs(aspectRatio-16.0/9.0) < tolerance {
		return "16:9", nil
	}
	if math.Abs(aspectRatio-9.0/16.0) < tolerance {
		return "9:16", nil
	}
	return "other", nil
}

func processVideoForFastStart(filePath string) (string, error) {
	// Moves the moov atom to the start of the video file.
	// The moov atom can be at the start or the end, usually at the end.
	// The moov atom contains video metadata
	// In the latter case, the video can only be played if the file is intact, which is imcompatible with streaming.
	// In the case of http requests with the Range header, the first request will usually get the moov atom to read the metadata
	outputFilePath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFilePath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return outputFilePath, nil
}
