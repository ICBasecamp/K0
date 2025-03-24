package s3

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

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
