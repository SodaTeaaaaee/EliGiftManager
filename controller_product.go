package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// ProductController exposes product Wails bindings.
type ProductController struct {
	uc app.ProductUseCase
}

func NewProductController() *ProductController {
	gdb := database.GetDB()
	masterRepo := infra.NewProductMasterRepository(gdb)
	productRepo := infra.NewProductRepository(gdb)
	waveRepo := infra.NewWaveRepository(gdb)
	return &ProductController{
		uc: app.NewProductUseCase(masterRepo, productRepo, waveRepo),
	}
}

// CreateProductMaster creates a new product master record.
func (c *ProductController) CreateProductMaster(input dto.CreateProductMasterInput) (*dto.ProductMasterDTO, error) {
	ctx := appContext
	return c.uc.CreateProductMaster(ctx, input)
}

// ListProductMasters returns all product masters.
func (c *ProductController) ListProductMasters() ([]dto.ProductMasterDTO, error) {
	ctx := appContext
	return c.uc.ListProductMasters(ctx)
}

// UpdateProductMaster updates an existing product master.
func (c *ProductController) UpdateProductMaster(input dto.UpdateProductMasterInput) (*dto.ProductMasterDTO, error) {
	ctx := appContext
	return c.uc.UpdateProductMaster(ctx, input)
}

// SnapshotProductsForWave creates wave-scoped product snapshots from master IDs.
func (c *ProductController) SnapshotProductsForWave(input dto.SnapshotProductsInput) ([]dto.ProductDTO, error) {
	ctx := appContext
	return c.uc.SnapshotProductsForWave(ctx, input)
}

// ListProductsByWave returns all products snapshotted into a wave.
func (c *ProductController) ListProductsByWave(waveID uint) ([]dto.ProductDTO, error) {
	ctx := appContext
	return c.uc.ListProductsByWave(ctx, waveID)
}
