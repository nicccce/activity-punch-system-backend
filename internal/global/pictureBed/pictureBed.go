package pictureBed

import (
	sysconfig "activity-punch-system/config"
	"context"
	"fmt"
	"mime/multipart"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// PictureBed 图片上传工具类（S3 对象存储）
// 用于将图片上传到 S3 兼容对象存储服务，并返回图片访问路径

type PictureBed struct {
	// S3 连接信息
	Endpoint  string // 例如：http://127.0.0.1:9000（MinIO）
	Region    string // 例如：us-east-1
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool

	// 是否使用 Path-Style（S3 兼容服务/自定义域名常用）。默认 true。
	UsePathStyle bool

	// 可选：对外访问的基础 URL（若为空则使用 Endpoint）。
	// 例如：https://cdn.example.com 或 https://bucket.example.com
	BaseURL string

	// 可选：对象 Key 前缀，如：punch/
	Prefix string

	s3Client *s3.Client
	uploader *manager.Uploader
}

// NewPictureBed 创建图片床实例
//   - endpoint: S3 兼容服务地址（含 scheme）
//   - baseURL: 访问基础 URL（可为空，为空时使用 endpoint）
func NewPictureBed(endpoint, baseURL string) *PictureBed {
	cfg := sysconfig.Get().S3
	return &PictureBed{
		Endpoint:     endpoint,
		BaseURL:      baseURL,
		AccessKey:    cfg.AccessKey,
		SecretKey:    cfg.SecretAccessKey,
		UseSSL:       strings.HasPrefix(strings.ToLower(endpoint), "https://"),
		UsePathStyle: cfg.UsePathStyle,
		Bucket:       cfg.Bucket,
		Region:       cfg.Region,
		Prefix:       cfg.Prefix,
	}
}

// InitS3 初始化 S3 客户端（建议在启动时调用）
func (pb *PictureBed) InitS3(ctx context.Context) error {
	if pb.Endpoint == "" {
		return fmt.Errorf("s3 endpoint is empty")
	}
	if pb.Region == "" {
		pb.Region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(pb.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(pb.AccessKey, pb.SecretKey, "")),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				if service == s3.ServiceID {
					return aws.Endpoint{URL: pb.Endpoint, SigningRegion: pb.Region, HostnameImmutable: true}, nil
				}
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			}),
		),
	)
	if err != nil {
		return err
	}

	pb.s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = pb.UsePathStyle
	})
	pb.uploader = manager.NewUploader(pb.s3Client)
	return nil
}

// SaveImage 上传图片到对象存储并返回图片 URL
func (pb *PictureBed) SaveImage(fileHeader *multipart.FileHeader) (string, error) {
	if pb.s3Client == nil || pb.uploader == nil {
		if err := pb.InitS3(context.Background()); err != nil {
			return "", err
		}
	}
	if pb.Bucket == "" {
		return "", fmt.Errorf("s3 bucket is empty")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	key := path.Join(strings.Trim(pb.Prefix, "/"), filename)
	key = strings.TrimLeft(key, "/")

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		// 兜底：不强依赖外部库进行 sniff
		contentType = "application/octet-stream"
	}

	_, err = pb.uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(pb.Bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}

	base := strings.TrimRight(pb.BaseURL, "/")
	if base == "" {
		base = strings.TrimRight(pb.Endpoint, "/")
	}

	// 返回可访问 URL：优先走 path-style（base/bucket/key），方便 BaseURL 为自定义域名场景。
	if pb.UsePathStyle {
		return base + "/" + pb.Bucket + "/" + key, nil
	}
	// virtual-host 风格需要 baseURL 自行包含 bucket 域名，这里仅拼 key
	return base + "/" + key, nil
}
