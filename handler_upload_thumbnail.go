package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
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
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unabled to parse form file", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	mimeType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "issue getting mime type from media", err)
		return
	}
	if !(mimeType == "image/jpeg" || mimeType == "image/png") {
		respondWithError(w, http.StatusBadRequest, "unaccepted media type submitted", errors.New("bad media type"))
	}

	fileData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to read image data", err)
		return
	}

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "something wrong with getting video data, are you the right user?", err)
		return
	}

	//* random id
	by := make([]byte, 32)
	_, err = rand.Read(by)
	if err != nil {
		fmt.Println("error getting random numberr: ", err)
		return
	}
	newUrl := base64.RawURLEncoding.EncodeToString(by)

	//* local /assets method
	extensionArr := strings.Split(mediaType, "/")
	thumbnailPath := newUrl + "." + extensionArr[len(extensionArr)-1]
	fp := filepath.Join(cfg.assetsRoot, thumbnailPath)

	fileOpen, err := os.Create(fp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating new file", err)
		return
	}
	defer fileOpen.Close()

	_, err = io.Copy(fileOpen, bytes.NewReader(fileData))
	// _, err = fileOpen.Write(fileData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "issue writing to file", err)
		return
	}

	fullUrl := "http://localhost:8091/" + fp
	videoData.ThumbnailURL = &fullUrl
	err = cfg.db.UpdateVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "issue with updating video", err)
	}

	respondWithJSON(w, http.StatusOK, videoData)
}
