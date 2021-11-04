package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ParsePath 解析路径，返回绝对路径
//
// basePath 当 path 为相对路径时必填
//
// 绝对路径        /xxx/xx => /xxx/xx
//
// 相对路径        xxx/xx  => <程序运行目录>/xxx/xx
//
// 相对路径        ./xx    => <程序运行目录>/xx
//
// 用户目录相对路径 ~/xx    => <用户家目录>/xx
//
func ParsePath(basePath, path string) (absPath string, err error) {
	if filepath.IsAbs(path) {
		return path, err
	}

	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		absPath = filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
		return absPath, nil
	}

	if !filepath.IsAbs(basePath) {
		return "", fmt.Errorf("basePath: %v must be absolute path", basePath)
	}

	return filepath.Join(basePath, path), nil
}

// IsFileExists 检查文件是否存在
func IsFileExists(path string) (isExists bool, err error) {
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return false, nil
	}

	if err == nil {
		return true, nil
	}

	return false, fmt.Errorf("failed to check path %s, %v", path, err)
}

type PathType string

var (
	PathTypeFile   PathType = "file"
	PathTypeFolder PathType = "folder"
)

// GetPathType 获取路径文件类型：file | folder
func GetPathType(path string) (fileType PathType, err error) {
	fstat, err := os.Stat(path)

	if err != nil {
		return "", fmt.Errorf("failed to get path type %v: %v", path, err)
	}

	if fstat.IsDir() {
		return PathTypeFolder, nil
	}

	return PathTypeFile, nil
}
