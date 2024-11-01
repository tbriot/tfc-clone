package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	DEFAULT_S3_BUCKET_NAME   = "tfc-configuration-files"
	DEFAULT_TMP_DOWNLOAD_DIR = "/tmp"
)

type S3TfConfigProvider struct {
	s3Client               *s3.Client
	bucketName             string
	defaultTempDownloadDir string
}

// Constructor
func newS3TfConfigProvider(cfg aws.Config) *S3TfConfigProvider {
	bucketName := DEFAULT_S3_BUCKET_NAME // create addressable variable from unadressable constant
	defaultTempDownloadDir := DEFAULT_TMP_DOWNLOAD_DIR
	return &S3TfConfigProvider{
		s3Client:               s3.NewFromConfig(cfg),
		bucketName:             bucketName,
		defaultTempDownloadDir: defaultTempDownloadDir,
	}
}

// Setters
func (p *S3TfConfigProvider) WithS3BucketName(bucketName string) {
	p.bucketName = bucketName
}

// Implementing the TfConfigProvider interface
func (p *S3TfConfigProvider) DownloadTfConfig(ctx context.Context, versionId string, path string) error {
	defer timeTrack(time.Now(), "download-tf-config")

	// Download S3 object
	objectKey := versionId
	result, err := p.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("couldn't get s3 object, bucketName=%v, objectKey=%v: %w\n", p.bucketName, objectKey, err)
	}
	defer result.Body.Close()

	// Create local file
	localFilepath := filepath.Join(DEFAULT_TMP_DOWNLOAD_DIR, objectKey)
	file, err := os.Create(localFilepath)
	if err != nil {
		return fmt.Errorf("couldn't create local file=%v: %w\n", localFilepath, err)
	}
	defer file.Close()

	// Write S3 object into local file
	body, err := io.ReadAll(result.Body)
	if err != nil {
		return fmt.Errorf("couldn't read s3 object body, objectKey=%v: %w\n", objectKey, err)
	}

	if _, err = file.Write(body); err != nil {
		return fmt.Errorf("couldn't write to local file=%v: %w\n", localFilepath, err)
	}

	err = unzipFile(localFilepath, path)

	return err
}

func unzipFile(filepath string, path string) error {
	defer timeTrack(time.Now(), "unzip-tf-config")

	// Create target dir if it does not exist
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return fmt.Errorf("error while creating directory=%v: %w", path, err)
	}

	// untar terraform config zip file
	cmd := exec.Command("tar", "-xf", filepath, "--strip-components=1", "-C", path)
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("error while unzipping tf config package, filepath=%v, ouput_path=%v: %w",
			filepath,
			path,
			err)
	}
	return nil
}
