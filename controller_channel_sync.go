package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// ChannelSyncController exposes channel-sync Wails bindings.
type ChannelSyncController struct {
	channelSyncUC   app.ChannelSyncUseCase
	channelSyncRepo domain.ChannelSyncRepository
}

func NewChannelSyncController() *ChannelSyncController {
	gdb := db.GetDB()
	channelSyncRepo := infra.NewChannelSyncRepository(gdb)
	shipmentRepo := infra.NewShipmentRepository(gdb)
	supplierRepo := infra.NewSupplierOrderRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	return &ChannelSyncController{
		channelSyncUC:   app.NewChannelSyncUseCase(channelSyncRepo, shipmentRepo, supplierRepo, fulfillRepo),
		channelSyncRepo: channelSyncRepo,
	}
}

// CreateChannelSyncJob creates a channel sync job with its items.
func (c *ChannelSyncController) CreateChannelSyncJob(input dto.CreateChannelSyncJobInput) (dto.ChannelSyncJobDTO, error) {
	job, items, err := c.channelSyncUC.CreateChannelSyncJob(input)
	if err != nil {
		return dto.ChannelSyncJobDTO{}, err
	}

	result := domainToChannelSyncJobDTO(job)
	result.Items = make([]dto.ChannelSyncItemDTO, len(items))
	for i, it := range items {
		result.Items[i] = domainToChannelSyncItemDTO(&it)
	}
	return result, nil
}

// ListChannelSyncJobsByWave lists all channel sync jobs for a given wave.
func (c *ChannelSyncController) ListChannelSyncJobsByWave(waveID uint) ([]dto.ChannelSyncJobDTO, error) {
	jobs, err := c.channelSyncRepo.ListJobsByWave(waveID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ChannelSyncJobDTO, len(jobs))
	for i, j := range jobs {
		jobDTO := domainToChannelSyncJobDTO(&j)
		items, err := c.channelSyncRepo.ListItemsByJob(j.ID)
		if err != nil {
			return nil, err
		}
		jobDTO.Items = make([]dto.ChannelSyncItemDTO, len(items))
		for k, it := range items {
			jobDTO.Items[k] = domainToChannelSyncItemDTO(&it)
		}
		result[i] = jobDTO
	}
	return result, nil
}

func domainToChannelSyncJobDTO(j *domain.ChannelSyncJob) dto.ChannelSyncJobDTO {
	if j == nil {
		return dto.ChannelSyncJobDTO{}
	}
	return dto.ChannelSyncJobDTO{
		ID:                   j.ID,
		WaveID:               j.WaveID,
		IntegrationProfileID: j.IntegrationProfileID,
		Direction:            j.Direction,
		Status:               j.Status,
		BasisHistoryNodeID:   j.BasisHistoryNodeID,
		BasisProjectionHash:  j.BasisProjectionHash,
		BasisPayloadSnapshot: j.BasisPayloadSnapshot,
		RequestPayload:       j.RequestPayload,
		ResponsePayload:      j.ResponsePayload,
		ErrorMessage:         j.ErrorMessage,
		StartedAt:            j.StartedAt,
		FinishedAt:           j.FinishedAt,
		CreatedAt:            j.CreatedAt,
		UpdatedAt:            j.UpdatedAt,
	}
}

func domainToChannelSyncItemDTO(it *domain.ChannelSyncItem) dto.ChannelSyncItemDTO {
	if it == nil {
		return dto.ChannelSyncItemDTO{}
	}
	return dto.ChannelSyncItemDTO{
		ID:                 it.ID,
		ChannelSyncJobID:   it.ChannelSyncJobID,
		FulfillmentLineID:  it.FulfillmentLineID,
		ShipmentID:         it.ShipmentID,
		ExternalDocumentNo: it.ExternalDocumentNo,
		ExternalLineNo:     it.ExternalLineNo,
		TrackingNo:         it.TrackingNo,
		CarrierCode:        it.CarrierCode,
		Status:             it.Status,
		ErrorMessage:       it.ErrorMessage,
		CreatedAt:          it.CreatedAt,
		UpdatedAt:          it.UpdatedAt,
	}
}
