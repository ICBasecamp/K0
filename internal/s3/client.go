package s3

import (
	"context"
	"fmt"
	"io"
	"os"

	"archive/tar"
	"path/filepath"

	// "github.com/docker/docker/pkg/archive"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"

	_ "github.com/joho/godotenv/autoload"
)

type S3Client struct {
	cli *s3.Client
	ctx context.Context
}

func CreateS3Client() (*S3Client, error) {

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	cli := s3.NewFromConfig(cfg)

	return &S3Client{cli: cli, ctx: context.TODO()}, nil
}

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

	uploader := manager.NewUploader(sc.cli)

	_, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    &key,
		Body:   pr,
	})
	return err
}


// debugging
func (sc *S3Client) ListObjects() ([]types.Object, error) {
	bucketName := os.Getenv("AWS_S3_BUCKET")
	if bucketName == "" {
		return nil, fmt.Errorf("AWS_S3_BUCKET environment variable is not set")
	}

	fmt.Printf("Listing objects from bucket: %s\n", bucketName)

	objects, err := sc.cli.ListObjectsV2(sc.ctx, &s3.ListObjectsV2Input{
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

// func (sc *S3Client) GetDockerBuildContext(key string) (io.ReadCloser, error) {
// 	bucketName := os.Getenv("AWS_S3_BUCKET")
// 	if bucketName == "" {
// 		return nil, fmt.Errorf("AWS_S3_BUCKET environment variable is not set")
// 	}



// 	buildContext, err := archive.TarWithOptions(buildContextPath, &archive.TarOptions{})
// 	if err != nil {
// 		return nil, fmt.Errorf("error creating tar: %w", err)
// 	}

// 	return buildContext, nil

// }
