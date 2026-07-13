package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
)

// GetCustomerFulfillmentHistory returns every fulfillment line belonging to
// the given customer profile across all waves, enriched with wave/product/
// shipment context (plan 5.4). It fills the CustomerDetail placeholder that
// previously had no cross-reference query exposed in the bridge.
func (c *CustomerProfileController) GetCustomerFulfillmentHistory(customerProfileID uint) ([]dto.CustomerFulfillmentHistoryRowDTO, error) {
	ctx := appContext
	uc := app.NewCustomerFulfillmentHistoryUseCase(database.GetDB())
	return uc.GetCustomerFulfillmentHistory(ctx, customerProfileID)
}
