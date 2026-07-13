package main

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	"gorm.io/gorm"
)

// MarkSupplierOrderSubmitted transitions a supplier order from draft to
// submitted, recording the factory-assigned external order number and the
// submission timestamp (plan 5.2 / 3.3.4).
func (c *ExportController) MarkSupplierOrderSubmitted(input dto.MarkSupplierOrderSubmittedInput) (dto.SupplierOrderDTO, error) {
	ctx := appContext

	var updated dto.SupplierOrderDTO
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		lineWriter := infra.NewSupplierOrderLineWriter(tx)
		lifecycleUC := app.NewSupplierOrderLifecycleUseCase(repos.SupplierRepo, lineWriter)

		order, err := lifecycleUC.MarkSupplierOrderSubmitted(ctx, input)
		if err != nil {
			return err
		}
		updated = domainToSupplierOrderDTO(order)
		return nil
	})
	if err != nil {
		return dto.SupplierOrderDTO{}, err
	}
	return updated, nil
}

// RecordSupplierOrderAcceptance transitions a supplier order from submitted
// to accepted, recording the factory-accepted quantity for each line (plan
// 5.2 / 3.3.4).
func (c *ExportController) RecordSupplierOrderAcceptance(input dto.RecordSupplierOrderAcceptanceInput) (dto.SupplierOrderDTO, error) {
	ctx := appContext

	var updated dto.SupplierOrderDTO
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		lineWriter := infra.NewSupplierOrderLineWriter(tx)
		lifecycleUC := app.NewSupplierOrderLifecycleUseCase(repos.SupplierRepo, lineWriter)

		order, err := lifecycleUC.RecordSupplierOrderAcceptance(ctx, input)
		if err != nil {
			return err
		}
		updated = domainToSupplierOrderDTO(order)
		return nil
	})
	if err != nil {
		return dto.SupplierOrderDTO{}, err
	}
	return updated, nil
}

// GenerateSupplierOrderFile writes the downloadable factory order file for
// the given supplier order and returns its output path (plan 3.3.4: file
// generation moved up to the factory step; the file embeds each line's ID
// and batch number for later shipment-return reconciliation).
func (c *ExportController) GenerateSupplierOrderFile(orderID uint) (dto.SupplierOrderFileResultDTO, error) {
	ctx := appContext

	exportsDir, err := service.ResolveExportsDir()
	if err != nil {
		return dto.SupplierOrderFileResultDTO{}, fmt.Errorf("resolve exports dir: %w", err)
	}

	fileWriter := app.NewSupplierOrderFileWriter(c.supplierRepo, exportsDir)
	return fileWriter.GenerateSupplierOrderFile(ctx, orderID)
}
