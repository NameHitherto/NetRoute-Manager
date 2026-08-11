//go:build windows

package store

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// documentsDir 返回当前用户的文档目录。
// 优先通过 Windows Known Folder API 获取,可正确处理 OneDrive 等目录重定向;
// 失败时回退到 <用户主目录>\Documents。
func documentsDir() (string, error) {
	dir, err := windows.KnownFolderPath(windows.FOLDERID_Documents, windows.KF_FLAG_DEFAULT)
	if err == nil && dir != "" {
		return dir, nil
	}
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		if err != nil {
			return "", err
		}
		return "", homeErr
	}
	return filepath.Join(home, "Documents"), nil
}
