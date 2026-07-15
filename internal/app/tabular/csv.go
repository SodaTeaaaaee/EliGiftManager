package tabular

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func readCSV(path string, opts ReadOptions) (*Sheet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("open csv file %q: %w", path, err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("csv file %q is empty", path)
	}

	text, err := decodeCSVBytes(raw, opts.Encoding)
	if err != nil {
		return nil, fmt.Errorf("decode csv file %q: %w", path, err)
	}

	reader := csv.NewReader(strings.NewReader(text))
	// Ragged rows must not hard-fail — guard width manually at the Sheet layer.
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv file %q: %w", path, err)
	}
	sheet, err := splitHeaderRows(records, opts.HasHeader)
	if err != nil {
		return nil, fmt.Errorf("csv file %q: %w", path, err)
	}
	return sheet, nil
}

// decodeCSVBytes applies Encoding. "auto"/"" strips a UTF-8 BOM, accepts valid
// UTF-8, and otherwise falls back to GBK (common for CN Excel exports).
func decodeCSVBytes(raw []byte, encoding string) (string, error) {
	enc := strings.ToLower(strings.TrimSpace(encoding))
	if enc == "" {
		enc = "auto"
	}

	switch enc {
	case "utf-8", "utf8":
		raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
		if !utf8.Valid(raw) {
			return "", fmt.Errorf("content is not valid utf-8")
		}
		return string(raw), nil
	case "gbk", "gb18030":
		return decodeGBK(raw)
	case "auto":
		raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
		if utf8.Valid(raw) {
			return string(raw), nil
		}
		return decodeGBK(raw)
	default:
		return "", fmt.Errorf("unsupported csv encoding %q", encoding)
	}
}

func decodeGBK(raw []byte) (string, error) {
	decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), raw)
	if err != nil {
		return "", fmt.Errorf("gbk decode: %w", err)
	}
	return string(decoded), nil
}
