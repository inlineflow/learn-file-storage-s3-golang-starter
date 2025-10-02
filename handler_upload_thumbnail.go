package main

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"slices"
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

	const maxMemory = 10 * 1024 * 1024
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, 500, "Error parsing MIME media type", err)
		return
	}

	supportedTypes := []string{"image/jpeg", "image/png"}
	if !slices.Contains(supportedTypes, mediaType) {
		respondWithError(w, 400, "Error parsing MIME media type", errors.New("Unsupported MIME type"))
		return
	}
	// data, err := io.ReadAll(file)
	// if err != nil {
	// 	respondWithError(w, 500, "Error reading file", err)
	// 	return
	// }

	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 500, "Error reading video metadata from db", err)
		return
	}

	ext := strings.Split(mediaType, "/")[1]
	path := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", videoIDString, ext))

	f, err := os.Create(path)
	if err != err {
		respondWithError(w, 500, "Error creating file", err)
		return
	}

	_, err = io.Copy(f, file)
	if err != nil {
		respondWithError(w, 500, "Error writing to file", err)
		return
	}

	tURL := filepath.Join("/", path)
	dbVideo.ThumbnailURL = &tURL

	// // t := thumbnail{
	// // 	data:      buf,
	// // 	mediaType: mediaType,
	// // }
	// b64 := base64.StdEncoding.EncodeToString(data)
	//
	// thumbnailURL := fmt.Sprintf("data:%s;base64;%s", mediaType, b64)
	// dbVideo.ThumbnailURL = &thumbnailURL
	//
	cfg.db.UpdateVideo(dbVideo)
	//
	respondWithJSON(w, http.StatusOK, dbVideo)
}
