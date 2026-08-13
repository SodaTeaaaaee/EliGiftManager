// Package csvformula mitigates spreadsheet formula injection in CSV exports.
package csvformula

// Sanitize prefixes a single quote when cell starts with =, +, -, @, tab, or CR.
// Safe cells (including empty) are returned unchanged.
func Sanitize(cell string) string {
	if cell == "" {
		return cell
	}
	switch cell[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + cell
	default:
		return cell
	}
}

// SanitizeRow returns a new slice with Sanitize applied to each cell.
func SanitizeRow(cells []string) []string {
	if cells == nil {
		return nil
	}
	out := make([]string, len(cells))
	for i, cell := range cells {
		out[i] = Sanitize(cell)
	}
	return out
}
