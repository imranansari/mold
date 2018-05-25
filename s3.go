package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3Publisher provides method for publishing to s3
type S3Publisher struct{}

// Publish publish provided artifact to s3
func (s *S3Publisher) Publish(cfg S3Config) error {
	var cred *credentials.Credentials

	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		cred = credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, "")
		_, err := cred.Get()
		if err != nil {
			return fmt.Errorf("failed to get creds %v", err)
		}
	}

	isDir, err := isDirectory(cfg.Source)
	if err != nil {
		return err
	}

	if isDir {
		err = filepath.Walk(cfg.Source, func(path string, f os.FileInfo, err error) error {
			if !f.IsDir() {
				log.Println(path)
				err := s.publish(cfg, path, true, cred)
				if err != nil {
					return err
				}
			}
			return nil
		})
	} else {
		err = s.publish(cfg, cfg.Source, false, cred)
	}

	return err
}

func (*S3Publisher) publish(cfg S3Config, filepath string, isDir bool, credentials *credentials.Credentials) error {
	f, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", cfg.Source, err)
	}
	defer f.Close()

	awsC := &aws.Config{
		Region:      aws.String(cfg.Region),
		Credentials: credentials,
	}
	sess := session.Must(session.NewSession(awsC))
	uploader := s3manager.NewUploader(sess)

	target := cfg.Target
	if isDir {
		target = path.Join(cfg.Target, filepath)
	}

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(target),
		Body:   f,
	})

	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}

	log.Printf("[INFO] file uploaded to, %s\n", result.Location)

	return nil
}

func isDirectory(path string) (bool, error) {
	fd, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	switch mode := fd.Mode(); {
	case mode.IsDir():
		return true, nil
	case mode.IsRegular():
		return false, nil
	}
	return false, nil
}
