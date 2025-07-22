package pictureBed

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

// PictureBed 图片上传工具类
// 用于将图片保存到本地指定目录，并返回图片访问路径

type PictureBed struct {
	SaveDir string // 图片保存目录
	BaseURL string // 图片访问基础URL
}

// NewPictureBed 创建图片床实例
func NewPictureBed(saveDir, baseURL string) *PictureBed {
	return &PictureBed{
		SaveDir: saveDir,
		BaseURL: baseURL,
	}
}

// SaveImage 保存图片到本地并返回图片URL
func (pb *PictureBed) SaveImage(fileHeader *multipart.FileHeader) (string, error) {
	// 打开上传的文件
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 确保保存目录存在
	if err := os.MkdirAll(pb.SaveDir, os.ModePerm); err != nil {
		return "", err
	}

	// 生成唯一文件名
	ext := filepath.Ext(fileHeader.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(pb.SaveDir, filename)

	// 创建目标文件
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// 拷贝内容
	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}

	// 返回图片访问URL
	return pb.BaseURL + "/" + filename, nil
}
