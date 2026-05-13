package dto

type FulfillmentLineDTO struct {
	ID                        uint   `json:"id"`
	WaveID                    uint   `json:"waveId"`
	CustomerProfileID         uint   `json:"customerProfileId"`
	WaveParticipantSnapshotID uint   `json:"waveParticipantSnapshotId"`
	ProductID                 *uint  `json:"productId"`
	DemandDocumentID          *uint  `json:"demandDocumentId"`
	DemandLineID              *uint  `json:"demandLineId"`
	CustomerAddressID         *uint  `json:"customerAddressId"`
	Quantity                  int    `json:"quantity"`
	AllocationState           string `json:"allocationState"`
	AddressState              string `json:"addressState"`
	SupplierState             string `json:"supplierState"`
	ChannelSyncState          string `json:"channelSyncState"`
	LineReason                string `json:"lineReason"`
	ExtraData                 string `json:"extraData"`
	CreatedAt                 string `json:"createdAt"`
	UpdatedAt                 string `json:"updatedAt"`
}
