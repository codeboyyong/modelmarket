package httpapi

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"model-market/backend/internal/config"
)

func (a *App) storageProvider() string {
	if a.Config.ObjectStorageProvider == "s3" {
		return "s3"
	}
	return "local"
}

func (a *App) objectUploadURL(ctx context.Context, key, contentType string) (string, error) {
	if a.storageProvider() == "s3" {
		if a.ObjectStore == nil {
			return "", errors.New("S3 object store is not initialized")
		}
		return a.ObjectStore.PresignPut(ctx, key, contentType, time.Duration(a.Config.S3PresignMinutes)*time.Minute)
	}
	return assetDownloadURL(a.Config.AssetPublicURL, a.Config.AssetBucket, key), nil
}

func (a *App) objectDownloadURL(ctx context.Context, bucket, key string) (string, error) {
	if a.Config.AssetPublicURL != "" {
		return assetDownloadURL(a.Config.AssetPublicURL, bucket, key), nil
	}
	if a.storageProvider() == "s3" {
		if a.ObjectStore == nil {
			return "", errors.New("S3 object store is not initialized")
		}
		return a.ObjectStore.PresignGet(ctx, key, time.Duration(a.Config.S3PresignMinutes)*time.Minute)
	}
	return assetDownloadURL(a.Config.AssetPublicURL, bucket, key), nil
}

func (a *App) writeGeneratedObject(ctx context.Context, key string, content []byte, contentType string) (int64, error) {
	if a.storageProvider() == "s3" {
		if a.ObjectStore == nil {
			return 0, errors.New("S3 object store is not initialized")
		}
		if err := a.ObjectStore.Put(ctx, key, content, contentType); err != nil {
			return 0, err
		}
		return int64(len(content)), nil
	}
	localPath, err := a.objectStoragePath(key)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(localPath, content, 0o644); err != nil {
		return 0, err
	}
	return int64(len(content)), nil
}

func (a *App) deleteStoredObject(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	if a.storageProvider() == "s3" {
		if a.ObjectStore == nil {
			return errors.New("S3 object store is not initialized")
		}
		return a.ObjectStore.Delete(ctx, key)
	}
	path, err := a.objectStoragePath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

type ObjectStore interface {
	Put(context.Context, string, []byte, string) error
	Delete(context.Context, string) error
	PresignPut(context.Context, string, string, time.Duration) (string, error)
	PresignGet(context.Context, string, time.Duration) (string, error)
}

type s3ObjectStore struct {
	bucket  string
	client  *s3.Client
	presign *s3.PresignClient
}

func NewS3ObjectStore(ctx context.Context, cfg config.Config) (ObjectStore, error) {
	if cfg.AssetBucket == "" || cfg.S3Region == "" {
		return nil, errors.New("MM_ASSET_BUCKET and AWS_REGION are required for S3 storage")
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.S3Region))
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.UsePathStyle = cfg.S3ForcePathStyle
		if cfg.S3Endpoint != "" {
			options.BaseEndpoint = aws.String(cfg.S3Endpoint)
		}
	})
	return &s3ObjectStore{bucket: cfg.AssetBucket, client: client, presign: s3.NewPresignClient(client)}, nil
}

func (s *s3ObjectStore) Put(ctx context.Context, key string, content []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), Body: bytes.NewReader(content), ContentType: aws.String(contentType)})
	return err
}

func (s *s3ObjectStore) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	return err
}

func (s *s3ObjectStore) PresignPut(ctx context.Context, key, contentType string, lifetime time.Duration) (string, error) {
	request, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), ContentType: aws.String(contentType)}, func(options *s3.PresignOptions) { options.Expires = lifetime })
	if err != nil {
		return "", err
	}
	return request.URL, nil
}

func (s *s3ObjectStore) PresignGet(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	request, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)}, func(options *s3.PresignOptions) { options.Expires = lifetime })
	if err != nil {
		return "", err
	}
	return request.URL, nil
}
