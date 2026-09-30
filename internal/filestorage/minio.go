package filestorage

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	particleassets "particleapp"
	"particleapp/internal/config"
)

const (
	DefaultImageKey = "default.jpg"
	DefaultVideoKey = "default.mp4"
)

// Storage — файлы услуги (изображение и короткое видео). Реализация — MinIO:
// имя файла сохраняется в поле модели, сам файл — в бакете.
type Storage interface {
	Upload(ctx context.Context, key string, file io.Reader, size int64, contentType string) error
	URL(key, fallbackKey string) string
}

type MinIO struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

func NewMinIO(cfg config.MinIOConfig) (*MinIO, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	storage := &MinIO{client: client, bucket: cfg.Bucket, publicURL: strings.TrimSuffix(cfg.PublicURL, "/")}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("minio: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("создание бакета %s: %w", cfg.Bucket, err)
		}
	}

	// Файлы услуг должны отдаваться клиенту по прямой ссылке без авторизации.
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, cfg.Bucket)
	if err := client.SetBucketPolicy(ctx, cfg.Bucket, policy); err != nil {
		log.Printf("minio: не удалось открыть бакет %s на чтение: %v", cfg.Bucket, err)
	}
	if err := storage.ensureDefaultObject(ctx, DefaultImageKey, "resources/static/media/default.jpg", "image/jpeg"); err != nil {
		return nil, err
	}
	if err := storage.ensureDefaultObject(ctx, DefaultVideoKey, "resources/static/media/default.mp4", "video/mp4"); err != nil {
		return nil, err
	}

	return storage, nil
}

// ensureDefaultObject один раз копирует встроенную заглушку в MinIO. Поэтому
// URL для записей без собственного изображения или видео не возвращает 404.
func (m *MinIO) ensureDefaultObject(ctx context.Context, key, assetPath, contentType string) error {
	if _, err := m.client.StatObject(ctx, m.bucket, key, minio.StatObjectOptions{}); err == nil {
		return nil
	} else if response := minio.ToErrorResponse(err); response.Code != "NoSuchKey" && response.Code != "NoSuchObject" && response.StatusCode != 404 {
		return fmt.Errorf("проверка файла-заглушки %s: %w", key, err)
	}

	file, err := particleassets.MediaFiles.Open(assetPath)
	if err != nil {
		return fmt.Errorf("открытие файла-заглушки %s: %w", key, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("размер файла-заглушки %s: %w", key, err)
	}
	if err := m.Upload(ctx, key, file, info.Size(), contentType); err != nil {
		return fmt.Errorf("загрузка файла-заглушки %s: %w", key, err)
	}
	return nil
}

func (m *MinIO) Upload(ctx context.Context, key string, file io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, m.bucket, key, file, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("загрузка %s в minio: %w", key, err)
	}
	return nil
}

// URL — публичная ссылка на файл. Если ключ не задан, отдаётся файл-заглушка,
// чтобы карточка услуги всегда содержала рабочую ссылку.
func (m *MinIO) URL(key, fallbackKey string) string {
	if key == "" {
		key = fallbackKey
	}
	return m.publicURL + "/" + key
}

func (m *MinIO) Bucket() string {
	return m.bucket
}
