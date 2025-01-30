package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	client := s3.NewPresignClient(s3Client)
	f := s3.WithPresignExpires(expireTime)
	params := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}
	req, err := client.PresignGetObject(context.TODO(), params, f)
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	url := video.VideoURL
	if url == nil {
		return video, nil
	}
	bucketAndKey := strings.Split(*url, ",")
	if len(bucketAndKey) != 2 {
		return video, errors.New("invalid videoURL")
	}
	bucket, key := bucketAndKey[0], bucketAndKey[1]
	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Minute*5)
	if err != nil {
		return video, err
	}
	video.VideoURL = &presignedURL
	return video, nil
}
