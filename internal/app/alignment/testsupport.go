package alignment

// PreviewRow is one parsed row as the template-testing surface shows it: the
// produced line number, the source data row it came from, the semantic values,
// and the duplicate-detection fingerprint the import would compute.
type PreviewRow struct {
	LineNo      int
	SourceRow   int
	Values      map[string]string
	Fingerprint string
}

// TemplatePreview is the result of running a template against sample data
// without touching any store: the first rows plus every issue found, with the
// counts needed to judge the mapping beyond the truncated sample.
type TemplatePreview struct {
	Rows   []PreviewRow
	Issues []ParseIssue
	// TotalRows is how many rows the parse produced before limit truncation.
	TotalRows int
	// DroppedRows counts source rows (or split candidates) the parse
	// discarded because a Required key was empty or the splitSkuQuantity blob
	// could not be expanded.
	DroppedRows int
}

// TestTemplate parses sample data with the given spec and returns up to limit
// rows plus all issues. It is the same Parse path the import use cases run,
// so a passing template test predicts a passing import.
func TestTemplate(data []byte, format string, spec TemplateSpec, limit int) (TemplatePreview, error) {
	rows, issues, stats, err := parseWithStats(data, format, spec)
	if err != nil {
		return TemplatePreview{}, err
	}
	total := len(rows)
	if limit < 0 {
		limit = 0
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	preview := make([]PreviewRow, 0, len(rows))
	for _, r := range rows {
		preview = append(preview, PreviewRow{LineNo: r.LineNo, SourceRow: r.SourceRow, Values: r.Values, Fingerprint: r.Fingerprint})
	}
	if issues == nil {
		issues = []ParseIssue{}
	}
	return TemplatePreview{Rows: preview, Issues: issues, TotalRows: total, DroppedRows: stats.Dropped}, nil
}
