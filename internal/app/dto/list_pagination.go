package dto

type CustomerProfilePageFilterInput struct {
	Keyword            string `json:"keyword"`
	Platform           string `json:"platform"`
	MissingAddressOnly bool   `json:"missingAddressOnly"`
	SortBy             string `json:"sortBy"`
	SortDir            string `json:"sortDir"`
	Limit              int    `json:"limit"`
	Offset             int    `json:"offset"`
}

type CustomerProfilePageResult struct {
	Items      []CustomerProfileDTO `json:"items"`
	TotalCount int                  `json:"totalCount"`
}

type ProductMasterPageFilterInput struct {
	Keyword      string   `json:"keyword"`
	ProductKinds []string `json:"productKinds"`
	ArchivedOnly bool     `json:"archivedOnly"`
	SortBy       string   `json:"sortBy"`
	SortDir      string   `json:"sortDir"`
	Limit        int      `json:"limit"`
	Offset       int      `json:"offset"`
}

type ProductMasterPageResult struct {
	Items      []ProductMasterDTO `json:"items"`
	TotalCount int                `json:"totalCount"`
}

type ShipmentByWavePageFilterInput struct {
	WaveID  uint   `json:"waveId"`
	SortBy  string `json:"sortBy"`
	SortDir string `json:"sortDir"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
}

type ShipmentPageResult struct {
	Items      []ShipmentDTO `json:"items"`
	TotalCount int           `json:"totalCount"`
}

type DemandInboxPageResult struct {
	Items      []DemandInboxRowDTO `json:"items"`
	TotalCount int                 `json:"totalCount"`
}

func NormalizeListPagination(limit, offset int) (int, int) {
	if limit < 1 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
