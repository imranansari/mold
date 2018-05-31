package main

import (
	"fmt"
	"os"
	"strings"
)

const (
	awsAccessKey     = "AWS_ACCESS_KEY_ID"
	awsSecretKey     = "AWS_SECRET_ACCESS_KEY"
	awsDefaultRegion = "AWS_DEFAULT_REGION"
)

// S3Config describes s3 artifact
type S3Config struct {
	Target    string `yaml: "target, omitempty"`
	Source    string `yaml: "source, omitempty"`
	Bucket    string `yaml: "bucket, omitempty"`
	AccessKey string `yaml: ", omitempty"`
	SecretKey string `yaml: ", omitempty"`
	Region    string `yaml: region, omitempty`
}

// Validate checks if file path, bucket and region fields are not empty
func (sc *S3Config) Validate() error {
	if sc.Source == "" {
		return fmt.Errorf("s3 config without source is not allowed")
	}

	if sc.Target == "" {
		return fmt.Errorf("s3 config without target is not allowed")
	}

	if sc.Bucket == "" {
		return fmt.Errorf("s3 config without bucket is not allowed")
	}

	if sc.Region == "" {
		return fmt.Errorf("s3 config without region is not allowed")
	}

	return nil
}

// ReplacePlaceholders replaces placeholders in Target and Source
func (sc *S3Config) ReplacePlaceholders(placeholder, value string) {
	sc.Target = strings.Replace(sc.Target, placeholder, value, -1)
	sc.Source = strings.Replace(sc.Source, placeholder, value, -1)
}

// LoadCredentialsFromEnv loads access, private keys and region from Environment
func (sc *S3Config) LoadCredentialsFromEnv() {
	if sc.AccessKey == "" {
		sc.AccessKey = os.Getenv(awsAccessKey)
	}

	if sc.SecretKey == "" {
		sc.SecretKey = os.Getenv(awsSecretKey)
	}

	if sc.Region == "" {
		sc.Region = os.Getenv(awsDefaultRegion)
	}
}
