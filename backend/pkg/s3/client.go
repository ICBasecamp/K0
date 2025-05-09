package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"archive/tar"
	"path/filepath"

	// "github.com/docker/docker/pkg/archive"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	_ "github.com/joho/godotenv/autoload"
)

type S3Client struct {
	client *s3.Client
	bucket string
}

func CreateS3Client() (*S3Client, error) {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	return &S3Client{
		client: client,
		bucket: os.Getenv("AWS_S3_BUCKET"),
	}, nil
}

// TarAndUploadToS3 creates a tar archive of the build context and uploads it to S3
func (sc *S3Client) TarAndUploadToS3(key, dir string) error {
	bucketName := os.Getenv("AWS_S3_BUCKET")
	if bucketName == "" {
		return fmt.Errorf("AWS_S3_BUCKET environment variable is not set")
	}

	pr, pw := io.Pipe()
	tw := tar.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer tw.Close()
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel(dir, path)
			if relPath == "." {
				return nil
			}
			hdr, _ := tar.FileInfoHeader(info, "")
			hdr.Name = relPath
			tw.WriteHeader(hdr)

			if info.Mode().IsRegular() {
				f, _ := os.Open(path)
				defer f.Close()
				io.Copy(tw, f)
			}
			return nil
		})
	}()

	uploader := manager.NewUploader(sc.client)

	_, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    &key,
		Body:   pr,
	})
	return err
}

func (sc *S3Client) GetDockerBuildContext(key string) (io.ReadCloser, error) {

	bucketName := os.Getenv("AWS_S3_BUCKET")
	if bucketName == "" {
		return nil, fmt.Errorf("AWS_S3_BUCKET environment variable is not set")
	}

	BuildContext, err := sc.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting object: %w", err)
	}

	return BuildContext.Body, nil
}

// debugging
func (sc *S3Client) ListObjects() ([]types.Object, error) {
	bucketName := os.Getenv("AWS_S3_BUCKET")
	if bucketName == "" {
		return nil, fmt.Errorf("AWS_S3_BUCKET environment variable is not set")
	}

	fmt.Printf("Listing objects from bucket: %s\n", bucketName)

	objects, err := sc.client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return nil, fmt.Errorf("error listing objects: %w", err)
	}

	for i, obj := range objects.Contents {
		fmt.Printf("%d. Key: %s, Size: %d bytes, LastModified: %s\n",
			i+1,
			*obj.Key,
			obj.Size,
			obj.LastModified.Format("2006-01-02 15:04:05"))
	}

	return objects.Contents, nil
}

// DownloadFromS3 downloads a build context from S3
func (s *S3Client) DownloadFromS3(key string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	return result.Body, nil
}

// FileExists checks if a file exists in S3
func (s *S3Client) FileExists(key string) (bool, error) {
	_, err := s.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if error is NotFound
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NotFound" {
			return false, nil
		}
		return false, fmt.Errorf("error checking file existence: %w", err)
	}
	return true, nil
}
