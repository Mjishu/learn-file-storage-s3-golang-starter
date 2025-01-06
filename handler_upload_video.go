package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIdString := r.PathValue("videoID")
	videoId, err := uuid.Parse(videoIdString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "couldnt find JWT", err)
		return
	}

	_, err = auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not validate JWT", err)
		return
	}

	videoData, err := cfg.db.GetVideo(videoId)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "you are not the correct user", err)
		return
	}

	maxUpload := 1 << 30
	r.ParseMultipartForm(int64(maxUpload))

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not find the video element", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	mimeType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not find media type", err)
		return
	}
	if mimeType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "video was not of correct type", err)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to read file", err)
		return
	}

	_, err = io.Copy(tempFile, bytes.NewReader(fileData))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not copy file", err)
	}

	//random32 byte key
	by := make([]byte, 32)
	_, err = rand.Read(by)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not make random key", err)
		return
	}
	key := base64.RawURLEncoding.EncodeToString(by) + ".mp4"

	tempFile.Seek(0, io.SeekStart)

	input := &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &key,
		Body:        tempFile,
		ContentType: &mimeType,
	}
	_, err = cfg.s3Client.PutObject(context.TODO(), input)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not store file in s3", err)
		return
	}

	videoURL := "https://" + cfg.s3Bucket + ".s3." + cfg.s3Region + ".amazonaws.com/" + key
	videoData.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "issue updating video", err)
	}

	respondWithJSON(w, http.StatusOK, videoData)
}
