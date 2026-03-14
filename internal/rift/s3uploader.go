package rift

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// AWSS3Uploader implements S3Uploader using the AWS SDK v2.
// It is used for both AWS S3 and S3-compatible endpoints (e.g. Yandex Object Storage).
type AWSS3Uploader struct {
	client *s3.Client
	bucket string
}

// NewS3UploaderFromConfig creates an AWSS3Uploader from a S3Config.
// When S3Config.Endpoint is set, the client is pointed at that endpoint
// (useful for Yandex Object Storage or MinIO).
func NewS3UploaderFromConfig(cfg S3Config) S3Uploader {
	return &AWSS3Uploader{
		client: buildS3Client(cfg),
		bucket: cfg.Bucket,
	}
}

func buildS3Client(cfg S3Config) *s3.Client {
	opts := []func(*awsconfig.LoadOptions) error{}
	if cfg.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.Region))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		// Return a client that will fail on use — errors surface at call time.
		return s3.NewFromConfig(aws.Config{})
	}

	s3Opts := []func(*s3.Options){}
	if cfg.Endpoint != "" {
		endpoint := cfg.Endpoint
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = &endpoint
			o.UsePathStyle = true // required for most S3-compatible endpoints
		})
	}

	return s3.NewFromConfig(awsCfg, s3Opts...)
}

// Upload writes r to the given key in the configured bucket.
func (u *AWSS3Uploader) Upload(ctx context.Context, key string, r io.Reader, size int64) error {
	_, err := u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(u.bucket),
		Key:           aws.String(key),
		Body:          r,
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return fmt.Errorf("s3 upload %s: %w", key, err)
	}
	return nil
}

// Download retrieves the object at key and writes it to w.
func (u *AWSS3Uploader) Download(ctx context.Context, key string, w io.Writer) error {
	out, err := u.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3 download %s: %w", key, err)
	}
	defer out.Body.Close()

	if _, err := io.Copy(w, out.Body); err != nil {
		return fmt.Errorf("s3 download read %s: %w", key, err)
	}
	return nil
}

// Exists reports whether the object at key exists in the bucket.
func (u *AWSS3Uploader) Exists(ctx context.Context, key string) (bool, error) {
	_, err := u.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// AWS SDK returns a NoSuchKey / 404 error — treat as not found.
		return false, nil
	}
	return true, nil
}
