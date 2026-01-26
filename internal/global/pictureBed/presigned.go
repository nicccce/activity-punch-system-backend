package pictureBed

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// PresignedUploadRequest 预签名上传请求参数
type PresignedUploadRequest struct {
	Filename    string // 原始文件名
	ContentType string // 文件 MIME 类型
	ExpiresIn   int64  // 过期时间（秒），默认 15 分钟
}

// PresignedUploadResponse 预签名上传响应
type PresignedUploadResponse struct {
	UploadURL string            `json:"upload_url"` // 预签名上传 URL
	FileKey   string            `json:"file_key"`   // 对象存储中的文件 key
	FileURL   string            `json:"file_url"`   // 上传成功后的访问 URL
	ExpiresAt time.Time         `json:"expires_at"` // 过期时间
	Method    string            `json:"method"`     // HTTP 方法（通常是 PUT）
	Headers   map[string]string `json:"headers"`    // 需要在上传时携带的 Headers
}

// GeneratePresignedUploadURL 生成预签名上传 URL
// 允许前端直接上传文件到 S3，无需经过后端中转
func (pb *PictureBed) GeneratePresignedUploadURL(ctx context.Context, req PresignedUploadRequest) (*PresignedUploadResponse, error) {
	// 确保 S3 客户端已初始化
	if pb.s3Client == nil {
		if err := pb.InitS3(ctx); err != nil {
			return nil, fmt.Errorf("初始化 S3 客户端失败: %w", err)
		}
	}

	// 验证必要参数
	if pb.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket 未配置")
	}
	if req.Filename == "" {
		return nil, fmt.Errorf("文件名不能为空")
	}

	// 设置默认过期时间（15 分钟）
	if req.ExpiresIn <= 0 {
		req.ExpiresIn = 900 // 15 分钟
	}

	// 生成唯一的文件名（时间戳 + 原始扩展名）
	ext := strings.ToLower(path.Ext(req.Filename))
	uniqueFilename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)

	// 构建完整的对象 key（包含前缀）
	key := path.Join(strings.Trim(pb.Prefix, "/"), uniqueFilename)
	key = strings.TrimLeft(key, "/")

	// 设置默认 Content-Type
	contentType := req.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 创建预签名客户端
	presignClient := s3.NewPresignClient(pb.s3Client)

	// 生成预签名 PUT 请求
	presignedReq, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(pb.Bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(req.ExpiresIn) * time.Second
	})

	if err != nil {
		return nil, fmt.Errorf("生成预签名 URL 失败: %w", err)
	}

	// 构建访问 URL
	base := strings.TrimRight(pb.BaseURL, "/")
	if base == "" {
		base = strings.TrimRight(pb.Endpoint, "/")
	}

	var fileURL string
	if pb.UsePathStyle {
		fileURL = base + "/" + pb.Bucket + "/" + key
	} else {
		fileURL = base + "/" + key
	}

	// 构建响应
	response := &PresignedUploadResponse{
		UploadURL: presignedReq.URL,
		FileKey:   key,
		FileURL:   fileURL,
		ExpiresAt: time.Now().Add(time.Duration(req.ExpiresIn) * time.Second),
		Method:    presignedReq.Method,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
	}

	// 添加预签名请求中的其他 Headers
	for k, v := range presignedReq.SignedHeader {
		if len(v) > 0 {
			response.Headers[k] = v[0]
		}
	}

	return response, nil
}

// GeneratePresignedDownloadURL 生成预签名下载 URL（可选功能）
// 用于访问私有对象
func (pb *PictureBed) GeneratePresignedDownloadURL(ctx context.Context, key string, expiresIn int64) (string, error) {
	if pb.s3Client == nil {
		if err := pb.InitS3(ctx); err != nil {
			return "", fmt.Errorf("初始化 S3 客户端失败: %w", err)
		}
	}

	if expiresIn <= 0 {
		expiresIn = 3600 // 默认 1 小时
	}

	presignClient := s3.NewPresignClient(pb.s3Client)

	presignedReq, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(pb.Bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(expiresIn) * time.Second
	})

	if err != nil {
		return "", fmt.Errorf("生成预签名下载 URL 失败: %w", err)
	}

	return presignedReq.URL, nil
}
