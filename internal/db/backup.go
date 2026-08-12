package db

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type databaseBackup struct {
	Path     string
	Checksum string
	Size     int64
}

func backupDatabaseBeforeMigration(dbPath string) (*databaseBackup, error) {
	info, err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat database before backup: %w", err)
	}
	if info.IsDir() || info.Size() == 0 {
		return nil, nil
	}

	backupDir := filepath.Join(filepath.Dir(dbPath), "backups")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return nil, fmt.Errorf("create database backup directory: %w", err)
	}
	base := fmt.Sprintf("%s.%s.bak", filepath.Base(dbPath), time.Now().UTC().Format("20060102T150405.000000000Z"))
	backupPath := filepath.Join(backupDir, base)

	type copiedFile struct {
		suffix   string
		checksum string
		size     int64
	}
	var copied []copiedFile
	for _, suffix := range []string{"", "-wal", "-shm"} {
		source := dbPath + suffix
		if _, err := os.Stat(source); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("stat database backup source %q: %w", source, err)
		}
		checksum, size, err := copyFileWithChecksum(source, backupPath+suffix)
		if err != nil {
			return nil, err
		}
		copied = append(copied, copiedFile{suffix: suffix, checksum: checksum, size: size})
	}

	sort.Slice(copied, func(i, j int) bool { return copied[i].suffix < copied[j].suffix })
	manifest := sha256.New()
	var totalSize int64
	for _, file := range copied {
		_, _ = io.WriteString(manifest, file.suffix+"\x00"+file.checksum+"\x00")
		totalSize += file.size
	}
	return &databaseBackup{Path: backupPath, Checksum: hex.EncodeToString(manifest.Sum(nil)), Size: totalSize}, nil
}

func copyFileWithChecksum(sourcePath, destinationPath string) (string, int64, error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return "", 0, fmt.Errorf("open database backup source %q: %w", sourcePath, err)
	}
	defer source.Close()

	destination, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", 0, fmt.Errorf("create database backup %q: %w", destinationPath, err)
	}
	keep := false
	defer func() {
		_ = destination.Close()
		if !keep {
			_ = os.Remove(destinationPath)
		}
	}()

	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(destination, hash), source)
	if err != nil {
		return "", 0, fmt.Errorf("copy database backup %q: %w", destinationPath, err)
	}
	if err := destination.Sync(); err != nil {
		return "", 0, fmt.Errorf("sync database backup %q: %w", destinationPath, err)
	}
	if err := destination.Close(); err != nil {
		return "", 0, fmt.Errorf("close database backup %q: %w", destinationPath, err)
	}
	checksum := hex.EncodeToString(hash.Sum(nil))
	verified, _, err := checksumFile(destinationPath)
	if err != nil {
		return "", 0, err
	}
	if verified != checksum {
		return "", 0, fmt.Errorf("database backup checksum mismatch for %q", destinationPath)
	}
	keep = true
	return checksum, size, nil
}

func checksumFile(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open file for checksum %q: %w", path, err)
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, fmt.Errorf("checksum file %q: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}
