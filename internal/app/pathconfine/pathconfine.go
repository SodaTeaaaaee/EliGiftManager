// Package pathconfine provides Windows-safe path confinement.
//
// Checks use filepath.Abs, volume-name equality, and a separator-aware
// prefix (not a bare strings.HasPrefix). Absolute paths, UNC paths, and
// ".." segments are rejected as join operands.
package pathconfine

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// LooksAbsolute reports drive-letter, UNC, and OS-absolute paths, including
// Windows forms such as `C:\...` and `\\server\share` even when GOOS is not
// windows (so template CoverDir values cannot smuggle those shapes).
func LooksAbsolute(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" {
		return false
	}
	n := filepath.FromSlash(p)
	if filepath.IsAbs(n) {
		return true
	}
	if isUNC(p) || isUNC(n) {
		return true
	}
	if drive, ok := windowsDrive(n); ok {
		return drive != ""
	}
	if drive, ok := windowsDrive(p); ok {
		return drive != ""
	}
	return false
}

// HasDotDot reports whether any path segment is "..".
func HasDotDot(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(filepath.FromSlash(p)), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

// JoinUnder joins rel onto root and requires the result to stay inside root.
// Absolute CoverDir/DetailDir values, UNC paths, and ".." are rejected.
func JoinUnder(root, rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("path is empty")
	}
	if LooksAbsolute(rel) {
		return "", fmt.Errorf("absolute or UNC path is not allowed")
	}
	if HasDotDot(rel) {
		return "", fmt.Errorf("path must not contain ..")
	}
	joined := filepath.Join(root, filepath.FromSlash(rel))
	return Confine(root, joined)
}

// UnderRoot returns the absolute candidate path if it is inside root.
// Argument order is (path, root) to match RevealInFolder call sites.
func UnderRoot(candidate, root string) (string, error) {
	return Confine(root, candidate)
}

// Confine returns the absolute candidate path if it is inside root.
func Confine(root, candidate string) (string, error) {
	if strings.TrimSpace(candidate) == "" {
		return "", fmt.Errorf("path is empty")
	}
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("root is empty")
	}
	if isUNC(candidate) && !isUNC(root) {
		return "", fmt.Errorf("UNC path is not allowed")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("abs root: %w", err)
	}
	absRoot = filepath.Clean(absRoot)

	absCand, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("abs path: %w", err)
	}
	absCand = filepath.Clean(absCand)

	if isUNC(absCand) && !isUNC(absRoot) {
		return "", fmt.Errorf("UNC path is not allowed")
	}
	if !sameVolume(absRoot, absCand) {
		return "", fmt.Errorf("path escapes root volume")
	}
	rel, err := filepath.Rel(absRoot, absCand)
	if err != nil {
		return "", fmt.Errorf("path escapes root: %w", err)
	}
	if rel == ".." || HasDotDot(rel) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path escapes root")
	}
	if !hasSepAwarePrefix(absCand, absRoot) {
		return "", fmt.Errorf("path escapes root")
	}
	return absCand, nil
}

func isUNC(p string) bool {
	p = strings.TrimSpace(p)
	if len(p) >= 2 && ((p[0] == '\\' && p[1] == '\\') || (p[0] == '/' && p[1] == '/')) {
		return true
	}
	vol := filepath.VolumeName(filepath.FromSlash(p))
	return strings.HasPrefix(vol, `\\`) || strings.HasPrefix(vol, `//`)
}

func windowsDrive(p string) (string, bool) {
	if len(p) < 2 || p[1] != ':' {
		return "", false
	}
	r := p[0]
	if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
		return "", false
	}
	return p[:2], true
}

func sameVolume(a, b string) bool {
	va := filepath.VolumeName(a)
	vb := filepath.VolumeName(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(va, vb)
	}
	return va == vb
}

func hasSepAwarePrefix(path, root string) bool {
	sep := string(os.PathSeparator)
	normPath, normRoot := path, root
	if runtime.GOOS == "windows" {
		normPath = strings.ToLower(path)
		normRoot = strings.ToLower(root)
	}
	if normPath == normRoot {
		return true
	}
	prefix := normRoot
	if !strings.HasSuffix(prefix, sep) {
		prefix += sep
	}
	return strings.HasPrefix(normPath, prefix)
}
