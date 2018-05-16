package main

import (
	"os"
	"testing"
)

func TestS3ConfigValidate(t *testing.T) {
	sc := &S3Config{
		File:   "test",
		Bucket: "buc",
		Region: "reg",
	}

	err := sc.Validate()
	if err != nil {
		t.Fatal(err)
	}

	sc.File = ""
	err = sc.Validate()
	if err == nil {
		t.Fatal("Expecting err not nil")
	}

	sc.File = "test"
	sc.Bucket = ""
	err = sc.Validate()
	if err == nil {
		t.Fatal("Expecting err not nil")
	}

	sc.Bucket = "test"
	sc.Region = ""
	err = sc.Validate()
	if err == nil {
		t.Fatal("Expecting err not nil")
	}
}

func Tests3LoadCredentialsFromEnt(t *testing.T) {
	sc := &S3Config{}

	os.Setenv(awsAccessKey, "test")
	os.Setenv(awsSecretKey, "test1")
	os.Setenv(awsDefaultRegion, "test2")

	sc.LoadCredentialsFromEnv()

	if sc.AccessKey != "test" {
		t.Fatalf("Expecting accesskey test, got: %s", sc.AccessKey)
	}

	if sc.SecretKey != "test1" {
		t.Fatalf("Expecting secretkey tes1, got: %s", sc.SecretKey)
	}

	if sc.Region != "test2" {
		t.Fatalf("Expecting region test2, got: %s", sc.Region)
	}
}
