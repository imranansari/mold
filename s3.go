package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Publisher struct{}

func (_ *S3Publisher) Publish(cfg S3Config) error {
	f, err := os.Open(cfg.File)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", cfg.File, err)
	}

	cred := credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, "")
	_, err = cred.Get()
	if err != nil {
		return fmt.Errorf("failed to get creds %v", err)
	}

	awsC := &aws.Config{
		Region:      aws.String(cfg.Region),
		Credentials: cred,
	}
	sess := session.Must(session.NewSession(awsC))
	uploader := s3manager.NewUploader(sess)

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(f.Name()),
		Body:   f,
	})

	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}

	log.Printf("[INFO] file uploaded to, %s\n", result.Location)

	return nil
}
