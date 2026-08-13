package controller

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

type MergeGovernanceController struct {
	uc *app.MergeGovernanceUseCase
}

func NewMergeGovernanceController() *MergeGovernanceController {
	gdb := database.GetDB()
	return &MergeGovernanceController{uc: app.NewMergeGovernanceUseCase(
		infra.NewMergeGovernanceRepository(gdb),
		infra.NewProfileRepository(gdb),
		infra.NewAddressRepository(gdb),
		infra.NewCustomerProfileOriginRepository(gdb),
	)}
}

func (c *MergeGovernanceController) GetMergePolicy() (*dto.MergePolicyDTO, error) {
	return c.uc.GetPolicy(appContext)
}

func (c *MergeGovernanceController) UpdateMergePolicy(input dto.UpdateMergePolicyInput) (*dto.MergePolicyDTO, error) {
	return c.uc.UpdatePolicy(appContext, input)
}

func (c *MergeGovernanceController) ScanMergeCandidates() (*dto.MergeScanRunDTO, error) {
	return c.uc.ScanMergeCandidates(appContext)
}

func (c *MergeGovernanceController) GetMergeScanRun(id uint) (*dto.MergeScanRunDTO, error) {
	return c.uc.GetScanRun(appContext, id)
}

func (c *MergeGovernanceController) ListMergeCandidates(status string) ([]dto.MergeCandidateDTO, error) {
	return c.uc.ListCandidates(appContext, status)
}

func (c *MergeGovernanceController) GetMergeCandidate(id uint) (*dto.MergeCandidateDTO, error) {
	return c.uc.GetCandidate(appContext, id)
}

func (c *MergeGovernanceController) DismissMergeCandidate(input dto.DismissMergeCandidateInput) error {
	return c.uc.DismissCandidate(appContext, input)
}
