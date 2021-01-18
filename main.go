package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/viper"
)

const (
	maxPartSize              = int64(5 * 1024 * 1024)
	maxRetries               = 3
	awsAccessKeyIDConfig     = "XUPLOADERID"
	awsSecretAccessKeyConfig = "XUPLOADERKEY"
	awsBucketNameConfig      = "XUPLOADERDIRECTORY"
)

func main() {

	var awsAccessKeyID, awsSecretAccessKey, awsBucketName string

	// Before running a .env file should be created with the environment variables
	viper.SetConfigFile(".env")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	awsAccessKeyID = viper.GetString(awsAccessKeyIDConfig)
	awsSecretAccessKey = viper.GetString(awsSecretAccessKeyConfig)
	awsBucketName = viper.GetString(awsBucketNameConfig)

	if awsAccessKeyID == "" {
		awsAccessKeyID = os.Getenv(awsAccessKeyIDConfig)
		awsSecretAccessKey = os.Getenv(awsSecretAccessKeyConfig)
		awsBucketName = os.Getenv(awsBucketNameConfig)
	}

	file, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error : %s\n", err.Error())
	}
	file = file + "/Downloads/tsetup.2.5.1.exe"

	// Configure AWS connection
	creds := credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, "")
	_, err = creds.Get()
	if err != nil {
		fmt.Printf("Bad credentials: %s", err.Error())
		os.Exit(1)
	}

	timeout, _ := time.ParseDuration("2h")
	sess := session.Must(session.NewSession(buildConfig(creds)))
	svc := s3.New(sess)

	ctx := context.Background()
	var cancelFn func()
	if timeout > 0 {
		ctx, cancelFn = context.WithTimeout(ctx, timeout)
	}

	if cancelFn != nil {
		defer cancelFn()
	}

	// Read the source file
	srcFile, err := os.Open(file)
	if err != nil {
		fmt.Printf("File not found %v\n%v\n", file, err)
		os.Exit(1)
	}

	defer srcFile.Close()

	fileInfo, _ := srcFile.Stat()
	fileSize := fileInfo.Size()
	buffer := make([]byte, fileSize)
	fileType := http.DetectContentType(buffer)
	srcFile.Read(buffer)

	_, fileName := filepath.Split(srcFile.Name())

	destPath := fileName

	// Prepare the multipart upload

	input := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(awsBucketName),
		Key:         aws.String(destPath),
		ContentType: aws.String(fileType),
	}

	resp, errSvc := svc.CreateMultipartUpload(input)
	if errSvc != nil {
		fmt.Println(errSvc.Error())
		os.Exit(1)
	}

	fmt.Printf("Starting upload %v with size: %v\n", *resp.UploadId, fileSize)

	// Execute the upload
	var curr, partLength int64
	var remaining = fileSize
	var completedParts []*s3.CompletedPart
	partNumber := 1

	for curr = 0; remaining != 0; curr += partLength {
		if remaining < maxPartSize {
			partLength = remaining
		} else {
			partLength = maxPartSize
		}

		completedPart, err := uploadPart(svc, resp, buffer[curr:curr+partLength], partNumber)
		if err != nil {
			fmt.Println(err.Error())
			err := abortMultipartUpload(svc, resp)
			if err != nil {
				fmt.Println(err.Error())
			}
			return
		}
		remaining -= partLength
		partNumber++
		completedParts = append(completedParts, completedPart)
	}

	completeResponse, err := completeMultipartUpload(svc, resp, completedParts)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Printf("Successfully uploaded file: %s\n", completeResponse.String())

}

func buildConfig(creds *credentials.Credentials) (config *aws.Config) {
	config = &aws.Config{Region: aws.String("us-east-1")}
	return config.WithCredentials(creds)
}

func uploadPart(svc *s3.S3, resp *s3.CreateMultipartUploadOutput, fileBytes []byte, partNumber int) (*s3.CompletedPart, error) {
	tryNum := 1
	partInput := &s3.UploadPartInput{
		Body:          bytes.NewReader(fileBytes),
		Bucket:        resp.Bucket,
		Key:           resp.Key,
		PartNumber:    aws.Int64(int64(partNumber)),
		UploadId:      resp.UploadId,
		ContentLength: aws.Int64(int64(len(fileBytes))),
	}
	for tryNum <= maxRetries {
		uploadResult, err := svc.UploadPart(partInput)
		if err != nil {
			if tryNum == maxRetries {
				if aerr, ok := err.(awserr.Error); ok {
					return nil, aerr
				}
				return nil, err
			}
			tryNum++
			fmt.Printf("Retrying to upload part #%v\n", partNumber)
		} else {
			fmt.Printf("Uploaded part #%v\n", partNumber)
			return &s3.CompletedPart{
				ETag:       uploadResult.ETag,
				PartNumber: aws.Int64(int64(partNumber)),
			}, nil
		}
	}
	return nil, nil
}

func abortMultipartUpload(svc *s3.S3, resp *s3.CreateMultipartUploadOutput) error {
	fmt.Println("Aborting multipart upload for upload id#" + *resp.UploadId)
	abortInput := &s3.AbortMultipartUploadInput{
		Bucket:   resp.Bucket,
		Key:      resp.Key,
		UploadId: resp.UploadId,
	}
	_, err := svc.AbortMultipartUpload(abortInput)
	return err
}

func completeMultipartUpload(svc *s3.S3, resp *s3.CreateMultipartUploadOutput, parts []*s3.CompletedPart) (*s3.CompleteMultipartUploadOutput, error) {

	completedInput := &s3.CompleteMultipartUploadInput{
		Bucket:   resp.Bucket,
		Key:      resp.Key,
		UploadId: resp.UploadId,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: parts,
		},
	}
	return svc.CompleteMultipartUpload(completedInput)
}

func loadEnvironmentVariables() {

}
