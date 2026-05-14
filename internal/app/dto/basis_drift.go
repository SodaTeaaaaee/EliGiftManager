package dto

type BasisDriftSignalDTO struct {
	BasisKind         string   `json:"basisKind"`
	BasisDriftStatus  string   `json:"basisDriftStatus"`
	ReviewRequirement string   `json:"reviewRequirement"`
	DriftReasonCodes  []string `json:"driftReasonCodes"`
}
