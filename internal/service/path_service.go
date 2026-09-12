package service

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	appDataDirName     = "EliGiftManager"
	dataDirName        = "data"
	assetsDirName      = "assets"
	tempDirName        = "tmp"
	portableMarkerName = ".portable"
	exportsSubDirName  = "exports"
)

// dataDirEnv carries the environment inputs of the data-directory decision so
// resolveDataDirCandidate can be tested with explicit dev/prod branches
// instead of build tags or environment variables.
type dataDirEnv struct {
	dev           bool
	workingDir    func() (string, error)
	executable    func() (string, error)
	userConfigDir func() (string, error)
	stat          func(string) (fs.FileInfo, error)
}

func defaultDataDirEnv() dataDirEnv {
	return dataDirEnv{
		dev:           isDevBuild(),
		workingDir:    os.Getwd,
		executable:    os.Executable,
		userConfigDir: os.UserConfigDir,
		stat:          os.Stat,
	}
}

// dataDirEnvSource is the environment factory ResolveDataDir reads from.
// Tests in this package may swap it to pin the dev/prod branch without
// relying on build tags or environment variables.
var dataDirEnvSource = defaultDataDirEnv

// ResolveDataDir 三级单选：
// 1. dev 构建（无 production 构建标签，即 wails3 dev 与 go test）→ 工作目录/data
// 2. exe 同级有 .portable 占位文件 → exe/data（便携模式）
// 3. 兜底 → os.UserConfigDir()/EliGiftManager/data（系统安装）
//
// 目录不存在时自动 MkdirAll。
func ResolveDataDir() (string, error) {
	dir, err := resolveDataDirCandidate(dataDirEnvSource())
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("resolve data dir: mkdir %q: %w", dir, err)
	}
	return dir, nil
}

func resolveDataDirCandidate(env dataDirEnv) (string, error) {
	if env.dev {
		wd, err := env.workingDir()
		if err != nil {
			return "", fmt.Errorf("resolve data dir: %w", err)
		}
		return filepath.Join(wd, dataDirName), nil
	}

	if execPath, err := env.executable(); err == nil {
		execDir := filepath.Dir(execPath)
		if _, statErr := env.stat(filepath.Join(execDir, portableMarkerName)); statErr == nil {
			return filepath.Join(execDir, dataDirName), nil
		}
	}

	uc, err := env.userConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve data dir: %w", err)
	}
	return filepath.Join(uc, appDataDirName, dataDirName), nil
}

// ResolveAssetsDir 返回 data/assets/ 目录。
func ResolveAssetsDir() (string, error) {
	dataDir, err := ResolveDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(dataDir, assetsDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("resolve assets dir: mkdir %q: %w", dir, err)
	}
	return dir, nil
}

// ResolveExportsDir returns data/exports/, the root for document_export executors.
func ResolveExportsDir() (string, error) {
	dataDir, err := ResolveDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(dataDir, exportsSubDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("resolve exports dir: mkdir %q: %w", dir, err)
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
		return "", fmt.Errorf("resolve temp dir: mkdir %q: %w", dir, err)
	}
	return dir, nil
}
