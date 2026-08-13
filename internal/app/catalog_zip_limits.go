package app

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
)

// Default catalog-zip resource limits (zip-bomb / zip-slip).
const (
	defaultCatalogZipMaxEntries    = 10_000
	defaultCatalogZipMaxFileBytes  = int64(32 << 20)  // 32 MiB
	defaultCatalogZipMaxTotalBytes = int64(256 << 20) // 256 MiB
	defaultCatalogZipMaxHashBytes  = int64(32 << 20)  // 32 MiB
)

type catalogZipLimits struct {
	MaxEntries    int
	MaxFileBytes  int64
	MaxTotalBytes int64
	MaxHashBytes  int64
}

func defaultCatalogZipLimits() catalogZipLimits {
	return catalogZipLimits{
		MaxEntries:    defaultCatalogZipMaxEntries,
		MaxFileBytes:  defaultCatalogZipMaxFileBytes,
		MaxTotalBytes: defaultCatalogZipMaxTotalBytes,
		MaxHashBytes:  defaultCatalogZipMaxHashBytes,
	}
}

func (d *catalogImportDeps) catalogZipLimits() catalogZipLimits {
	def := defaultCatalogZipLimits()
	if d == nil || d.zipLimits == nil {
		return def
	}
	lim := *d.zipLimits
	if lim.MaxEntries <= 0 {
		lim.MaxEntries = def.MaxEntries
	}
	if lim.MaxFileBytes <= 0 {
		lim.MaxFileBytes = def.MaxFileBytes
	}
	if lim.MaxTotalBytes <= 0 {
		lim.MaxTotalBytes = def.MaxTotalBytes
	}
	if lim.MaxHashBytes <= 0 {
		lim.MaxHashBytes = def.MaxHashBytes
	}
	return lim
}

// copyLimited copies src to dst with a hard cap. It never uses unbounded io.Copy.
func copyLimited(dst io.Writer, src io.Reader, limit int64) (int64, error) {
	if limit <= 0 {
		return 0, fmt.Errorf("invalid copy limit")
	}
	n, err := io.Copy(dst, io.LimitReader(src, limit+1))
	if n > limit {
		return n, fmt.Errorf("uncompressed size exceeds limit of %d bytes", limit)
	}
	return n, err
}

func hashSHA256Limited(r io.Reader, limit int64) (string, int64, error) {
	h := sha256.New()
	n, err := copyLimited(h, r, limit)
	if err != nil {
		return "", n, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// zipCatalogAssetMetadata records member path/hash/size without retaining
// binary payloads. Hashing is capped (LimitReader) so a zip bomb cannot force
// unbounded reads. Tabular members are excluded because their rows are stored.
func zipCatalogAssetMetadata(path string, limits catalogZipLimits) ([]map[string]string, error) {
	if limits.MaxHashBytes <= 0 {
		limits.MaxHashBytes = defaultCatalogZipMaxHashBytes
	}
	if limits.MaxEntries <= 0 {
		limits.MaxEntries = defaultCatalogZipMaxEntries
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open zip for evidence metadata: %w", err)
	}
	defer reader.Close()
	if len(reader.File) > limits.MaxEntries {
		return nil, fmt.Errorf("zip has %d entries, exceeds limit of %d", len(reader.File), limits.MaxEntries)
	}
	out := make([]map[string]string, 0)
	for _, member := range reader.File {
		if member.FileInfo().IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(member.Name))
		if ext == ".csv" || ext == ".xlsx" || ext == ".xls" {
			continue
		}
		if member.UncompressedSize64 > uint64(limits.MaxHashBytes) {
			return nil, fmt.Errorf("zip member %q exceeds hash size limit", member.Name)
		}
		r, err := member.Open()
		if err != nil {
			return nil, err
		}
		sum, _, hashErr := hashSHA256Limited(r, limits.MaxHashBytes)
		closeErr := r.Close()
		if hashErr != nil {
			return nil, fmt.Errorf("hash zip member %q: %w", member.Name, hashErr)
		}
		if closeErr != nil {
			return nil, closeErr
		}
		out = append(out, map[string]string{
			"member": member.Name,
			"path":   filepath.ToSlash(member.Name),
			"sha256": sum,
			"size":   fmt.Sprint(member.UncompressedSize64),
		})
	}
	return out, nil
}

func isTabularZipMember(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".csv" || ext == ".xlsx" || ext == ".xls"
}

// osLongPath prefixes Windows paths with \\?\ so extracted catalog trees
// longer than MAX_PATH can still be created. Other OSes are unchanged.
func osLongPath(p string) string {
	if runtime.GOOS != "windows" {
		return p
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	abs = filepath.Clean(abs)
	if strings.HasPrefix(abs, `\\?\`) {
		return abs
	}
	if strings.HasPrefix(abs, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(abs, `\\`)
	}
	return `\\?\` + abs
}
