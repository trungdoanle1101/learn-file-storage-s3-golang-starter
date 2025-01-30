package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const uploadLimit = 1 << 30
	const fileFormKey = "video"
	const tempFileName = "tubely-upload.mp4"
	r.Body = http.MaxBytesReader(w, r.Body, uploadLimit)
	videoID, err := uuid.Parse(r.PathValue("videoID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid videoID provided", err)
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

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You are not authorized to perform this action", nil)
		return
	}

	file, header, err := r.FormFile(fileFormKey)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get form file for video", err)
		return
	}

	defer file.Close()
	contentType := header.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse media type", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Please upload a valid .mp4 file", nil)
		return
	}

	tempFile, err := os.Create(tempFileName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temp file", err)
		return
	}
	defer os.Remove(tempFileName)
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy to temp file", err)
		return
	}

	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't reset file offset", err)
		return
	}

	fastStartFileName, err := processVideoForFastStart(tempFileName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create fast start video", err)
		return
	}
	defer os.Remove(fastStartFileName)

	fastStartFile, err := os.Open(fastStartFileName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't open fast start file", err)
		return
	}
	defer fastStartFile.Close()

	ar, err := getVideoAspectRatio(tempFileName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't obtain aspect ratio", err)
		return
	}

	arType := "other"
	if ar == "16:9" {
		arType = "landscape"
	} else if ar == "9:16" {
		arType = "portrait"
	}

	randomBytes := make([]byte, 32)
	_, err = rand.Read(randomBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't generate random bytes", err)
		return
	}

	objectKey := fmt.Sprintf("%s/%s.mp4", arType, hex.EncodeToString(randomBytes))
	s3PutObjectInput := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &objectKey,
		Body:        fastStartFile,
		ContentType: &mediaType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &s3PutObjectInput)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't put object to s3", err)
		return
	}

	// newURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, objectKey)
	newURL := fmt.Sprintf("%s,%s", cfg.s3Bucket, objectKey)
	video.VideoURL = &newURL
	err = cfg.db.UpdateVideo(video)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update videoURL", err)
		return
	}

	video, err = cfg.dbVideoToSignedVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't presign video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)

}
