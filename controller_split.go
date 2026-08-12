package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

type SplitController struct{ gdb *gorm.DB }

func NewSplitController() *SplitController {
	return &SplitController{gdb: database.GetDB()}
}

func (c *SplitController) PreviewCustomerSplit(input dto.CustomerSplitPreviewInput) (*dto.CustomerSplitPreviewResult, error) {
	executor := app.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(c.gdb))
	return executor.PreviewSplit(appContext, input)
}

func (c *SplitController) ExecuteCustomerSplit(input dto.ExecuteCustomerSplitInput) (*dto.ExecuteCustomerSplitResult, error) {
	var result *dto.ExecuteCustomerSplitResult
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		executor := app.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(tx))
		var executeErr error
		result, executeErr = executor.ExecuteSplit(appContext, input)
		return executeErr
	})
	return result, err
}

func (c *SplitController) ListCustomerSplitHistory(query dto.CustomerSplitHistoryQuery) (*dto.CustomerSplitHistoryPage, error) {
	useCase := app.NewCustomerSplitHistoryUseCase(infra.NewSplitExecutionStore(c.gdb))
	return useCase.ListSplitHistory(appContext, query)
}

func (c *SplitController) GetCustomerSplitHistory(splitID uint) (*dto.CustomerSplitHistoryDetail, error) {
	useCase := app.NewCustomerSplitHistoryUseCase(infra.NewSplitExecutionStore(c.gdb))
	return useCase.GetSplitHistory(appContext, splitID)
}
