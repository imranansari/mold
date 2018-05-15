package main

import (
	"fmt"
)

type S3Config struct {
	File      string `yaml: "file, omitempty"`
	Bucket    string `yaml: "bucket, omitempty"`
	AccessKey string `yaml: ", omitempty"`
	SecretKey string `yaml: ", omitempty"`
	Region    string `yaml: region, omitempty`
}

func (sc *S3Config) Validate() error {
	if sc.File == "" {
		return fmt.Errorf("s3 config without file is not allowed")
	}

	if sc.Bucket == "" {
		return fmt.Errorf("s3 config without bucket is not allowed")
	}

	if sc.Region == "" {
		return fmt.Errorf("s3 config without region is not allowed")
	}

	return nil
}
