package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// newAddressBatchUseCase self-constructs an AddressBatchUseCase from the DB singleton,
// matching AddressController's self-contained construction style (no gdb field on the
// controller itself).
func newAddressBatchUseCase() app.AddressBatchUseCase {
	gdb := database.GetDB()
	addressRepo := infra.NewAddressRepository(gdb)
	fulfillmentRepo := infra.NewFulfillmentRepository(gdb)
	return app.NewAddressBatchUseCase(addressRepo, fulfillmentRepo)
}

// BatchBindAddressToLines binds each entry's address to its fulfillment line, returning
// a per-entry result (partial-success semantics).
func (c *AddressController) BatchBindAddressToLines(entries []dto.BindAddressEntry) ([]dto.AddressBatchItemResult, error) {
	ctx := appContext
	uc := c.batchUC
	if uc == nil {
		uc = newAddressBatchUseCase()
	}
	return uc.BatchBindAddressToLines(ctx, entries)
}

// BindDefaultAddressesForWave binds the recipient's default address to every
// address-missing fulfillment line in the given wave.
func (c *AddressController) BindDefaultAddressesForWave(waveID uint) ([]dto.AddressBatchItemResult, error) {
	ctx := appContext
	uc := c.batchUC
	if uc == nil {
		uc = newAddressBatchUseCase()
	}
	return uc.BindDefaultAddressesForWave(ctx, waveID)
}
