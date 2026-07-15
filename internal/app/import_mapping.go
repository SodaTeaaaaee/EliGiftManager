package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// DocumentFieldDraft holds document.* values extracted from a mapped import row.
type DocumentFieldDraft struct {
	SourceCustomerRef string
	SourceDocumentNo  string
	DisplayName       string
}

// RecipientAddressDraft holds recipient.* values extracted from a mapped import row.
// A nil draft means no recipient.* keys were present on the row.
type RecipientAddressDraft struct {
	RecipientName string
	Phone         string
	Country       string
	Province      string
	City          string
	District      string
	AddressLine1  string
	AddressLine2  string
	PostalCode    string
	Label         string
	IsDefault     bool
}

// DemandImportMappedRow is one successfully mapped demand-import row spanning
// line.*, document.*, and optional recipient.* namespaces.
type DemandImportMappedRow struct {
	Line      *domain.DemandLine
	Document  DocumentFieldDraft
	Recipient *RecipientAddressDraft
}

// namespaceField splits "ns.field" dest keys. Unprefixed keys return ("", dest, true)
// so v1 demand-line fields keep working as the semantic default "line." namespace.
func namespaceField(dest string) (ns, field string, ok bool) {
	dest = strings.TrimSpace(dest)
	if dest == "" {
		return "", "", false
	}
	if i := strings.IndexByte(dest, '.'); i >= 0 {
		ns = dest[:i]
		field = dest[i+1:]
		if ns == "" || field == "" || strings.Contains(field, ".") {
			return "", "", false
		}
		return ns, field, true
	}
	return "", dest, true
}

// extractDocumentDraft reads document.* keys from an ApplyRow result.
func extractDocumentDraft(applied map[string]string) DocumentFieldDraft {
	var d DocumentFieldDraft
	for dest, val := range applied {
		ns, field, ok := namespaceField(dest)
		if !ok || ns != "document" {
			continue
		}
		switch field {
		case "source_customer_ref", "sourceCustomerRef":
			d.SourceCustomerRef = val
		case "source_document_no", "sourceDocumentNo":
			d.SourceDocumentNo = val
		case "display_name", "displayName":
			d.DisplayName = val
		}
	}
	return d
}

// extractRecipientDraft reads recipient.* keys. Returns nil when none are present
// so callers can skip address writes (downstream bind-default path).
func extractRecipientDraft(applied map[string]string) *RecipientAddressDraft {
	var d RecipientAddressDraft
	found := false
	for dest, val := range applied {
		ns, field, ok := namespaceField(dest)
		if !ok || ns != "recipient" {
			continue
		}
		found = true
		switch field {
		case "name", "recipient_name", "recipientName":
			d.RecipientName = val
		case "phone":
			d.Phone = val
		case "country":
			d.Country = val
		case "province":
			d.Province = val
		case "city":
			d.City = val
		case "district":
			d.District = val
		case "address_line1", "addressLine1":
			d.AddressLine1 = val
		case "address_line2", "addressLine2":
			d.AddressLine2 = val
		case "postal_code", "postalCode":
			d.PostalCode = val
		case "label":
			d.Label = val
		case "is_default", "isDefault":
			d.IsDefault = parseBoolish(val)
		}
	}
	if !found {
		return nil
	}
	return &d
}

func parseBoolish(val string) bool {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "1", "true", "yes", "y", "是":
		return true
	default:
		return false
	}
}

// mapAppliedToDemandLine builds a DemandLine from line.* (and unprefixed v1) keys.
func mapAppliedToDemandLine(applied map[string]string) (*domain.DemandLine, error) {
	line := &domain.DemandLine{}
	hasLineField := false
	for dest, val := range applied {
		field, ok := demandLineFieldName(dest)
		if !ok {
			continue
		}
		hasLineField = true
		if err := setDemandLineField(line, field, val); err != nil {
			return nil, fmt.Errorf("field %q: %w", dest, err)
		}
	}
	if !hasLineField {
		// Allow document/recipient-only rows only when explicitly mapped; demand import
		// still needs at least one line field to persist a DemandLine.
		return nil, fmt.Errorf("no line.* fields mapped on row")
	}
	return line, nil
}

// MapDemandImportRow applies mapping rules to an ordered row and splits namespaces.
// Mapping-dest warnings from ApplyRow are returned alongside the mapped row so
// callers can surface unknown-dest vocabulary issues without dropping values.
func MapDemandImportRow(row []string, headers []string, rules *TemplateMappingRules) (*DemandImportMappedRow, []string, error) {
	applied, warnings, err := ApplyRow(row, headers, rules)
	if err != nil {
		return nil, nil, err
	}
	line, err := mapAppliedToDemandLine(applied)
	if err != nil {
		return nil, warnings, err
	}
	return &DemandImportMappedRow{
		Line:      line,
		Document:  extractDocumentDraft(applied),
		Recipient: extractRecipientDraft(applied),
	}, warnings, nil
}

// MapDemandImportRowFromHeaderMap is the header-keyed convenience path used when
// the caller already holds map[string]string rows (frontend preview reuse).
func MapDemandImportRowFromHeaderMap(row map[string]string, rules *TemplateMappingRules) (*DemandImportMappedRow, []string, error) {
	if rules == nil {
		return nil, nil, fmt.Errorf("mapping rules are nil")
	}
	if rules.Mode == "positional" {
		return nil, nil, fmt.Errorf("positional mapping requires ordered row cells; use MapDemandImportRow")
	}
	headers := make([]string, 0, len(row))
	cells := make([]string, 0, len(row))
	for k, v := range row {
		headers = append(headers, k)
		cells = append(cells, v)
	}
	return MapDemandImportRow(cells, headers, rules)
}

// parseIntLoose parses a decimal integer, defaulting blank to 0.
func parseIntLoose(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}
