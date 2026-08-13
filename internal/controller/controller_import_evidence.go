package controller

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// ImportEvidenceController keeps ordinary list calls metadata-only. Raw PII is
// returned solely by the explicit GetImportRunDetail endpoint.
type ImportEvidenceController struct{ uc *app.ImportEvidenceUseCase }

func NewImportEvidenceController() *ImportEvidenceController {
	return &ImportEvidenceController{uc: app.NewImportEvidenceUseCase(infra.NewImportEvidenceRepository(db.GetDB()))}
}
func (c *ImportEvidenceController) ListImportRuns(limit int) ([]dto.ImportRunSummaryDTO, error) {
	return c.uc.ListRuns(appContext, limit)
}
func (c *ImportEvidenceController) ListImportRunsPage(input dto.ListImportRunsPageInput) (dto.ImportRunPageDTO, error) {
	return c.uc.ListRunsPage(appContext, input)
}
func (c *ImportEvidenceController) GetImportRunDetail(id uint) (dto.ImportRunDetailDTO, error) {
	return c.uc.GetRunDetail(appContext, id)
}
func (c *ImportEvidenceController) GetImportEvidenceRetention() (dto.ImportEvidenceRetentionDTO, error) {
	return c.uc.GetRetention(appContext)
}
func (c *ImportEvidenceController) SetImportEvidenceRetention(input dto.SetImportEvidenceRetentionInput) (dto.ImportEvidenceRetentionDTO, error) {
	return c.uc.SetRetention(appContext, input)
}
func (c *ImportEvidenceController) PruneExpiredImportEvidence() (map[string]int64, error) {
	runs, records, err := c.uc.PruneExpired(appContext)
	if err != nil {
		return nil, err
	}
	return map[string]int64{"runsDeleted": runs, "recordsDeleted": records}, nil
}
