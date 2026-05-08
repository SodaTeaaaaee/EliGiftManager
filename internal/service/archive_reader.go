package service

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractArchive detects the archive format by extension and extracts it
// to a temporary directory. Returns the extraction directory path.
func ExtractArchive(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".zip":
		return extractZip(path)
	case ".gz", ".tgz":
		return extractTarGz(path)
	case ".tar":
		return extractTar(path, nil)
	default:
		// Try tar.gz if .tgz or filename ends with .tar.gz
		base := strings.ToLower(filepath.Base(path))
		if strings.HasSuffix(base, ".tar.gz") {
			return extractTarGz(path)
		}
		if strings.HasSuffix(base, ".tar.bz2") || strings.HasSuffix(base, ".tar.xz") {
			return "", fmt.Errorf("extract archive: unsupported compression format for %q (supported: zip, tar, tar.gz, tgz)", path)
		}
		return "", fmt.Errorf("extract archive: unsupported format %q for file %q", ext, path)
	}
}

func extractZip(path string) (string, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("extract zip: open %q: %w", path, err)
	}
	defer archive.Close()

	extractDir, err := os.MkdirTemp("", "eligift-product-archive-*")
	if err != nil {
		return "", fmt.Errorf("extract zip: create temp dir: %w", err)
	}

	for _, f := range archive.File {
		if f.FileInfo().IsDir() {
			os.MkdirAll(filepath.Join(extractDir, f.Name), 0o755)
			continue
		}
		destPath := filepath.Join(extractDir, strings.ReplaceAll(f.Name, "*", "_"))
		cleanExtract, _ := filepath.Abs(extractDir)
		cleanDest, _ := filepath.Abs(destPath)
		if !strings.HasPrefix(cleanDest, cleanExtract+string(os.PathSeparator)) && cleanDest != cleanExtract {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		out, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			continue
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if copyErr != nil {
			continue
		}
	}
	return extractDir, nil
}

func extractTarGz(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("extract tar.gz: open %q: %w", path, err)
	}
	defer f.Close()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("extract tar.gz: gzip reader: %w", err)
	}
	defer gzReader.Close()

	return extractTar(path, gzReader)
}

func extractTar(path string, reader io.Reader) (string, error) {
	var r io.Reader
	var closer io.Closer
	if reader != nil {
		r = reader
	} else {
		f, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("extract tar: open %q: %w", path, err)
		}
		defer f.Close()
		r = f
		closer = f
	}
	_ = closer

	extractDir, err := os.MkdirTemp("", "eligift-product-archive-*")
	if err != nil {
		return "", fmt.Errorf("extract tar: create temp dir: %w", err)
	}

	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("extract tar: read header: %w", err)
		}

		destPath := filepath.Join(extractDir, strings.ReplaceAll(hdr.Name, "*", "_"))
		cleanExtract, _ := filepath.Abs(extractDir)
		cleanDest, _ := filepath.Abs(destPath)
		if !strings.HasPrefix(cleanDest, cleanExtract+string(os.PathSeparator)) && cleanDest != cleanExtract {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(destPath, 0o755)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				continue
			}
			out, err := os.Create(destPath)
			if err != nil {
				continue
			}
			_, copyErr := io.Copy(out, tr)
			out.Close()
			if copyErr != nil {
				continue
			}
		}
	}
	return extractDir, nil
}

// FindCSVInDir searches dir for a CSV file matching pattern (glob).
// Returns the full path of the first match.
func FindCSVInDir(dir, pattern string) (string, error) {
	if pattern == "" {
		pattern = "*.csv"
	}
	// Try exact pattern first
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err == nil && len(matches) > 0 {
		return matches[0], nil
	}
	// Fallback: recursive search
	globFiles, err := filepath.Glob(filepath.Join(dir, "**", "*.csv"))
	if err == nil && len(globFiles) > 0 {
		return globFiles[0], nil
	}
	// Last fallback: direct *.csv in root
	matches, _ = filepath.Glob(filepath.Join(dir, "*.csv"))
	if len(matches) > 0 {
		return matches[0], nil
	}
	return "", fmt.Errorf("no CSV file found in %q", dir)
}

// FindAllCSVsInDir recursively searches dir for all CSV files.
// Returns relative paths (from dir) of matching files.
func FindAllCSVsInDir(dir string) []string {
	var results []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".csv") {
			rel, _ := filepath.Rel(dir, path)
			if rel != "" {
				results = append(results, rel)
			}
		}
		return nil
	})
	return results
}

// ListArchiveDirs returns a summary of top-level directories and their
// file counts within an extracted archive directory.
// Deprecated: use ListArchiveDirTree for recursive directory enumeration.
func ListArchiveDirs(extractDir string) []ArchiveDirInfo {
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return nil
	}
	infos := make([]ArchiveDirInfo, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		subDir := filepath.Join(extractDir, e.Name())
		fileCount := 0
		filepath.Walk(subDir, func(_ string, fi os.FileInfo, _ error) error {
			if fi != nil && !fi.IsDir() {
				fileCount++
			}
			return nil
		})
		infos = append(infos, ArchiveDirInfo{Name: e.Name(), FileCount: fileCount})
	}
	return infos
}

type ArchiveDirInfo struct {
	Name      string `json:"name"`
	FileCount int    `json:"fileCount"`
}

// ArchiveDirNode is a recursive directory tree node for UI rendering.
type ArchiveDirNode struct {
	Name      string            `json:"name"`
	FileCount int               `json:"fileCount"`
	Children  []ArchiveDirNode  `json:"children,omitempty"`
}

// ListArchiveDirTree returns a recursive directory tree of all subdirectories
// within an extracted archive or source directory. Each node includes its
// direct file count (excluding subdirectory contents) and child directories.
func ListArchiveDirTree(extractDir string) []ArchiveDirNode {
	return walkDirTree(extractDir, extractDir)
}

func walkDirTree(root string, current string) []ArchiveDirNode {
	entries, err := os.ReadDir(current)
	if err != nil {
		return nil
	}
	nodes := make([]ArchiveDirNode, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		subDir := filepath.Join(current, e.Name())
		fileCount := 0
		children := walkDirTree(root, subDir)
		dirEntries, _ := os.ReadDir(subDir)
		for _, de := range dirEntries {
			if de != nil && !de.IsDir() {
				fileCount++
			}
		}
		// Relative path from extract root for display and matching.
		relPath, _ := filepath.Rel(root, subDir)
		nodes = append(nodes, ArchiveDirNode{
			Name:      filepath.ToSlash(relPath),
			FileCount: fileCount,
			Children:  children,
		})
	}
	return nodes
}
