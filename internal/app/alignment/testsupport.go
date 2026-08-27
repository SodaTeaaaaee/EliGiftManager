package alignment

// TemplatePreview is the result of running a template against sample data
// without touching any store: the first rows plus every issue found.
type TemplatePreview struct {
	Rows   []map[string]string
	Issues []ParseIssue
}

// TestTemplate parses sample data with the given spec and returns up to limit
// rows plus all issues. It is the same Parse path the import use cases run,
// so a passing template test predicts a passing import.
func TestTemplate(data []byte, format string, spec TemplateSpec, limit int) (TemplatePreview, error) {
	rows, issues, err := Parse(data, format, spec)
	if err != nil {
		return TemplatePreview{}, err
	}
	if limit < 0 {
		limit = 0
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	values := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		values = append(values, r.Values)
	}
	if issues == nil {
		issues = []ParseIssue{}
	}
	return TemplatePreview{Rows: values, Issues: issues}, nil
}
