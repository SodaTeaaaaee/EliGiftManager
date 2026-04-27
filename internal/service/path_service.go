package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveDataDir 三级探测：
// 1. exe 在系统 Temp → 工作目录/data（wails dev）
// 2. exe 同级有 .portable 占位文件 → exe/data（便携模式）
// 3. 兜底 → os.UserConfigDir()/EliGiftManager/data（系统安装）
// 目录不存在时自动 MkdirAll。
func ResolveDataDir() (string, error) {
	var dir string
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		if !strings.HasPrefix(execDir, os.TempDir()) {
			// Check .portable first
			if _, statErr := os.Stat(filepath.Join(execDir, ".portable")); statErr == nil {
				dir = filepath.Join(execDir, "data")
			} else {
				// System install
				uc, ucErr := os.UserConfigDir()
				if ucErr != nil {
					return "", fmt.Errorf("resolve data dir: %w", ucErr)
				}
				dir = filepath.Join(uc, "EliGiftManager", "data")
			}
		}
	}
	if dir == "" {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return "", fmt.Errorf("resolve data dir: %w", wdErr)
		}
		dir = filepath.Join(wd, "data")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("resolve data dir: mkdir %q: %w", dir, err)
	}
	return dir, nil
}

// ResolveAssetsDir 返回 data/assets/ 目录。
func ResolveAssetsDir() (string, error) {
	dataDir, err := ResolveDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(dataDir, "assets")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("resolve assets dir: %w", err)
	}
	return dir, nil
}
