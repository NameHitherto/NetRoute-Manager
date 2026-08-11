//go:build !windows

package store

import (
	"os"
	"path/filepath"
)

// documentsDir 返回当前用户的文档目录(非 Windows 平台使用 ~/Documents)。
func documentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents"), nil
}
