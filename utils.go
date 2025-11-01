package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func getVideoAspectRatio(filePath string) (string, error) {

	type fileDetails struct {
		Streams []struct {
			Width  int64 `json:"width"`
			Height int64 `json:"height"`
		} `json:"streams"`
	}

	stdoutErr, err := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath).CombinedOutput()
	if err != nil {
		log.Printf("command finished with error")
		return "", err
	}

	/*
		var b bytes.Buffer
		cmd.Stdout = &b
		if err := cmd.Run(); err != nil {
			log.Printf("command finished with error, %v", err)
			return "", err
		}*/

	var f fileDetails
	err = json.Unmarshal(stdoutErr, &f)
	if err != nil {
		log.Printf("error unmarshalling json, %v", err)
		return "", err
	}

	aspectRatio := f.Streams[0].Width / f.Streams[0].Height
	switch aspectRatio {
	case 1:
		return "16:9", nil
	case 0:
		return "9:16", nil
	default:
		return "other", nil
	}

}

func processVideoForFastStart(filePath string) (string, error) {
	outFilePath := fmt.Sprintf("%s.processing", filePath)

	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outFilePath)

	if err := cmd.Run(); err != nil {
		log.Printf("command finished with err")
		return "", err
	}

	return outFilePath, nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	req, err := presignClient.PresignGetObject(context.Background(),
		&s3.GetObjectInput{Bucket: &bucket,
			Key: &key}, s3.WithPresignExpires(expireTime))
	if err != nil {
		log.Printf("could not generate url")
		return "", err
	}

	return req.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	values := strings.Split(*video.VideoURL, ",")
	bucket := values[0]
	key := values[1]

	presignedURL, err := generatePresignedURL(cfg.s3client, bucket, key, 150*time.Second)
	if err != nil {
		log.Printf("error creating preseigned url")
		return database.Video{}, err
	}

	video.VideoURL = &presignedURL
	return video, nil
}
