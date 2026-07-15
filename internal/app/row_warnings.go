package app

import "fmt"

// rowWarningCollector accumulates row-prefixed, deduplicated mapping warnings
// (e.g. unknown mapping-dest vocabulary surfaced by ApplyRow/warnUnknownDests)
// across an import batch, so callers can attach them to a result DTO's
// Warnings field without dropping the row-level context or repeating an
// identical row+message pair.
type rowWarningCollector struct {
	seen  map[string]struct{}
	items []string
}

// add formats each warning as "row N: <warning>" (rowIndex is zero-based,
// matching the existing RowIndex/EntryIndex convention on error DTOs) and
// appends it if the exact same formatted message has not already been added.
func (c *rowWarningCollector) add(rowIndex int, warnings []string) {
	for _, w := range warnings {
		msg := fmt.Sprintf("row %d: %s", rowIndex, w)
		if c.seen == nil {
			c.seen = make(map[string]struct{})
		}
		if _, ok := c.seen[msg]; ok {
			continue
		}
		c.seen[msg] = struct{}{}
		c.items = append(c.items, msg)
	}
}

// warnings returns the accumulated messages as a non-nil slice (empty, not
// null, when there are none) so JSON-encoded result DTOs always carry a
// `"warnings": []` rather than `"warnings": null` — frontend call sites treat
// the field as always-array.
func (c *rowWarningCollector) warnings() []string {
	if c.items == nil {
		return []string{}
	}
	return c.items
}
