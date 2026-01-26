package tools

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
)

func FileExist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true // 文件存在
	}
	if os.IsNotExist(err) {
		return false // 文件不存在
	}
	// 其他错误，如权限问题等
	return false
}

func SearchFile(fileName string) (filePath string) {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	for {
		if _, err := filepath.Glob(filepath.Join(dir, fileName)); err == nil {
			return filepath.Join(filepath.Dir(dir), fileName)
		}
		dir = filepath.Dir(dir)
	}
}

const (
	ExcelContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
)

func SendStoredFile(c *gin.Context, path, displayName, contentType string) error {
	escaped := url.QueryEscape(displayName)

	c.Header("Content-Type", contentType)
	c.Header(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, escaped, escaped),
	)

	c.File(path)
	return nil
}
