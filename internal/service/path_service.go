package service

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	appDataDirName = "EliGiftManager"
	dataDirName    = "data"
	assetsDirName  = "assets"
	tempDirName    = "tmp"
)

// ResolveDataDir 三级单选：
// 1. Wails dev 环境变量存在 → 工作目录/data
// 2. exe 同级有 .portable 占位文件 → exe/data（便携模式）
// 3. 兜底 → os.UserConfigDir()/EliGiftManager/data（系统安装）
//
// 目录不存在时自动 MkdirAll。
func ResolveDataDir() (string, error) {
	dir, err := resolveDataDirCandidate()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("resolve data dir: mkdir %q: %w", dir, err)
	}
	return dir, nil
}

func resolveDataDirCandidate() (string, error) {
	if isWailsDevMode() {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve data dir: %w", err)
		}
		return filepath.Join(wd, dataDirName), nil
	}

	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		if _, statErr := os.Stat(filepath.Join(execDir, ".portable")); statErr == nil {
			return filepath.Join(execDir, dataDirName), nil
		}
	}

	uc, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve data dir: %w", err)
	}
	return filepath.Join(uc, appDataDirName, dataDirName), nil
}

func isWailsDevMode() bool {
	return os.Getenv("devserver") != "" || os.Getenv("frontenddevserverurl") != ""
}

// ResolveAssetsDir 返回 data/assets/ 目录。
func ResolveAssetsDir() (string, error) {
	dataDir, err := ResolveDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(dataDir, assetsDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("resolve assets dir: %w", err)
	}
	return dir, nil
}

// ResolveExportsDir returns data/exports/, the root for document_export executors.
func ResolveExportsDir() (string, error) {
	dataDir, err := ResolveDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(dataDir, "exports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("resolve exports dir: %w", err)
	}
	return dir, nil
}

// ResolveTempDir returns the app-managed temporary directory under data/tmp.
func ResolveTempDir() (string, error) {
	dataDir, err := ResolveDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(dataDir, tempDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("resolve temp dir: %w", err)
	}
	return dir, nil
}
