package controller

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// SnapshotProductsForWaveDetailed exposes the skip-detail variant of
// SnapshotProductsForWave (plan 5.3): for each requested product master ID it
// reports whether a new wave-scoped Product row was created or one already
// existed for that wave/platform/SKU and was skipped.
func (c *ProductController) SnapshotProductsForWaveDetailed(input dto.SnapshotProductsInput) (*dto.SnapshotProductsDetailedResult, error) {
	ctx := appContext
	gdb := database.GetDB()
	masterRepo := infra.NewProductMasterRepository(gdb)
	productRepo := infra.NewProductRepository(gdb)
	waveRepo := infra.NewWaveRepository(gdb)
	uc := app.NewProductSnapshotDetailUseCase(masterRepo, productRepo, waveRepo)
	return uc.SnapshotProductsForWaveDetailed(ctx, input)
}
