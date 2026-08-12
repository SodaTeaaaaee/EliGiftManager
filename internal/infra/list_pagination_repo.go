package infra

import (
	"context"
	"strings"
	"unicode"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type listPaginationRepository struct {
	db *gorm.DB
}

func NewListPaginationRepository(db *gorm.DB) *listPaginationRepository {
	return &listPaginationRepository{db: db}
}

func pageOrder(sortBy, sortDir string, whitelist map[string]string, idColumn string) string {
	column, ok := whitelist[sortBy]
	if !ok {
		column = idColumn
	}
	direction := "ASC"
	if sortDir == "desc" {
		direction = "DESC"
	}
	return column + " " + direction + ", " + idColumn + " ASC"
}

func (r *listPaginationRepository) ListCustomerProfilesPage(ctx context.Context, q domain.CustomerProfilePageQuery) ([]domain.CustomerProfile, int64, error) {
	query := r.db.WithContext(ctx).Model(&persistence.CustomerProfile{})
	if q.Platform != "" {
		query = query.Where(`EXISTS (SELECT 1 FROM customer_identities ci
			WHERE ci.customer_profile_id = customer_profiles.id AND ci.deleted_at IS NULL
			AND LOWER(ci.identity_platform) = LOWER(?))`, q.Platform)
	}
	if q.MissingAddressOnly {
		query = query.Where(`NOT EXISTS (SELECT 1 FROM customer_addresses ca
			WHERE ca.customer_profile_id = customer_profiles.id AND ca.deleted_at IS NULL)`)
	}
	if keyword := strings.TrimSpace(q.Keyword); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		predicate := `LOWER(customer_profiles.display_name) LIKE ? OR EXISTS
			(SELECT 1 FROM customer_identities ci WHERE ci.customer_profile_id = customer_profiles.id
			AND ci.deleted_at IS NULL AND LOWER(ci.identity_value) LIKE ?)`
		args := []any{like, like}
		if r.db.Migrator().HasTable("customer_name_observations") {
			predicate += ` OR EXISTS (SELECT 1 FROM customer_name_observations cno
				WHERE cno.customer_profile_id = customer_profiles.id AND cno.deleted_at IS NULL
				AND cno.is_active = true AND (LOWER(cno.name) LIKE ? OR cno.normalized_name LIKE ?))`
			args = append(args, like, "%"+normalizeNameSearch(keyword)+"%")
		}
		query = query.Where(predicate, args...)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := pageOrder(q.SortBy, q.SortDir, map[string]string{
		"displayName": "customer_profiles.display_name",
		"profileType": "customer_profiles.profile_type",
		"createdAt":   "customer_profiles.created_at",
	}, "customer_profiles.id")
	var rows []persistence.CustomerProfile
	if err := query.Order(order).Offset(q.Offset).Limit(q.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]domain.CustomerProfile, len(rows))
	for i := range rows {
		items[i] = *persistence.CustomerProfileToDomain(&rows[i])
	}
	return items, total, nil
}

func (r *listPaginationRepository) FindMatchedCustomerHistoricalNames(ctx context.Context, ids []uint, keyword string) (map[uint]string, error) {
	result := make(map[uint]string)
	keyword = strings.TrimSpace(keyword)
	if len(ids) == 0 || keyword == "" || !r.db.Migrator().HasTable("customer_name_observations") {
		return result, nil
	}
	type matchRow struct {
		CustomerProfileID uint
		Name              string
	}
	var rows []matchRow
	like := "%" + strings.ToLower(keyword) + "%"
	if err := r.db.WithContext(ctx).Table("customer_name_observations").
		Select("customer_profile_id, name").
		Where("customer_profile_id IN ? AND deleted_at IS NULL AND is_active = ?", ids, true).
		Where("LOWER(name) LIKE ? OR normalized_name LIKE ?", like, "%"+normalizeNameSearch(keyword)+"%").
		Order("last_seen_at DESC, id DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if _, exists := result[row.CustomerProfileID]; !exists {
			result[row.CustomerProfileID] = row.Name
		}
	}
	return result, nil
}

func normalizeNameSearch(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, strings.TrimSpace(value))
}

func (r *listPaginationRepository) ListCustomerIdentitiesByProfileIDs(ctx context.Context, ids []uint) ([]domain.CustomerIdentity, error) {
	if len(ids) == 0 {
		return []domain.CustomerIdentity{}, nil
	}
	var rows []persistence.CustomerIdentity
	if err := r.db.WithContext(ctx).Where("customer_profile_id IN ?", ids).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.CustomerIdentity, len(rows))
	for i := range rows {
		items[i] = *persistence.CustomerIdentityToDomain(&rows[i])
	}
	return items, nil
}

func (r *listPaginationRepository) ListCustomerAddressesByProfileIDs(ctx context.Context, ids []uint) ([]domain.CustomerAddress, error) {
	if len(ids) == 0 {
		return []domain.CustomerAddress{}, nil
	}
	var rows []persistence.CustomerAddress
	if err := r.db.WithContext(ctx).Where("customer_profile_id IN ?", ids).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.CustomerAddress, len(rows))
	for i := range rows {
		items[i] = *persistence.CustomerAddressToDomain(&rows[i])
	}
	return items, nil
}

func (r *listPaginationRepository) ListCustomerIdentityPlatforms(ctx context.Context) ([]string, error) {
	var platforms []string
	err := r.db.WithContext(ctx).Model(&persistence.CustomerIdentity{}).
		Joins("JOIN customer_profiles ON customer_profiles.id = customer_identities.customer_profile_id AND customer_profiles.deleted_at IS NULL").
		Distinct("customer_identities.identity_platform").
		Where("customer_identities.identity_platform <> ''").
		Order("customer_identities.identity_platform").
		Pluck("customer_identities.identity_platform", &platforms).Error
	return platforms, err
}

func (r *listPaginationRepository) ListProductMastersPage(ctx context.Context, q domain.ProductMasterPageQuery) ([]domain.ProductMaster, int64, error) {
	query := r.db.WithContext(ctx).Model(&persistence.ProductMaster{})
	if q.ArchivedOnly {
		query = query.Where("product_masters.archived = ?", true)
	}
	if keyword := strings.TrimSpace(q.Keyword); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		query = query.Where(`LOWER(product_masters.name) LIKE ? OR LOWER(product_masters.factory_sku) LIKE ?
			OR LOWER(product_masters.supplier_product_ref) LIKE ?`, like, like, like)
	}
	if len(q.ProductKinds) > 0 {
		query = query.Where("product_masters.product_kind IN ?", q.ProductKinds)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := pageOrder(q.SortBy, q.SortDir, map[string]string{
		"name":               "product_masters.name",
		"supplierPlatform":   "product_masters.supplier_platform",
		"factorySku":         "product_masters.factory_sku",
		"supplierProductRef": "product_masters.supplier_product_ref",
		"productKind":        "product_masters.product_kind",
		"archived":           "product_masters.archived",
	}, "product_masters.id")
	var rows []persistence.ProductMaster
	if err := query.Order(order).Offset(q.Offset).Limit(q.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]domain.ProductMaster, len(rows))
	for i := range rows {
		items[i] = *persistence.ProductMasterToDomain(&rows[i])
	}
	return items, total, nil
}

func (r *listPaginationRepository) ListDemandDocumentsPage(ctx context.Context, q domain.DemandInboxPageQuery) ([]domain.DemandDocument, int64, error) {
	query := r.db.WithContext(ctx).Model(&persistence.DemandDocument{})
	if q.DemandKind != "" {
		query = query.Where("demand_documents.kind = ?", q.DemandKind)
	}
	if q.IntegrationProfileID != nil {
		query = query.Where("demand_documents.integration_profile_id = ?", *q.IntegrationProfileID)
	}
	assignmentExists := `EXISTS (SELECT 1 FROM wave_demand_assignments wda
		WHERE wda.demand_document_id = demand_documents.id AND wda.deleted_at IS NULL)`
	if q.Assignment == "assigned" {
		query = query.Where(assignmentExists)
	} else if q.Assignment == "unassigned" {
		query = query.Where("NOT " + assignmentExists)
	}
	if q.WaveID != nil {
		query = query.Where(`EXISTS (SELECT 1 FROM wave_demand_assignments wda
			WHERE wda.demand_document_id = demand_documents.id AND wda.deleted_at IS NULL AND wda.wave_id = ?)`, *q.WaveID)
	}
	var total int64
	if err := query.Distinct("demand_documents.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := pageOrder(q.SortBy, q.SortDir, map[string]string{
		"kind":             "demand_documents.kind",
		"captureMode":      "demand_documents.capture_mode",
		"sourceChannel":    "demand_documents.source_channel",
		"sourceDocumentNo": "demand_documents.source_document_no",
		"createdAt":        "demand_documents.created_at",
	}, "demand_documents.id")
	var rows []persistence.DemandDocument
	if err := query.Order(order).Offset(q.Offset).Limit(q.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]domain.DemandDocument, len(rows))
	for i := range rows {
		items[i] = *persistence.DemandDocumentToDomain(&rows[i])
	}
	return items, total, nil
}

func (r *listPaginationRepository) ListDemandAssignmentsByDocumentIDs(ctx context.Context, ids []uint) ([]domain.WaveDemandAssignment, error) {
	if len(ids) == 0 {
		return []domain.WaveDemandAssignment{}, nil
	}
	var rows []persistence.WaveDemandAssignment
	if err := r.db.WithContext(ctx).Where("demand_document_id IN ?", ids).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.WaveDemandAssignment, len(rows))
	for i := range rows {
		items[i] = *persistence.WaveDemandAssignmentToDomain(&rows[i])
	}
	return items, nil
}

func (r *listPaginationRepository) ListDemandLinesByDocumentIDs(ctx context.Context, ids []uint) ([]domain.DemandLine, error) {
	if len(ids) == 0 {
		return []domain.DemandLine{}, nil
	}
	var rows []persistence.DemandLine
	if err := r.db.WithContext(ctx).Where("demand_document_id IN ?", ids).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.DemandLine, len(rows))
	for i := range rows {
		items[i] = *persistence.DemandLineToDomain(&rows[i])
	}
	return items, nil
}

func (r *listPaginationRepository) ListShipmentsPage(ctx context.Context, q domain.ShipmentByWavePageQuery) ([]domain.Shipment, int64, error) {
	query := r.db.WithContext(ctx).Model(&persistence.Shipment{}).
		Joins("JOIN supplier_orders ON supplier_orders.id = shipments.supplier_order_id AND supplier_orders.deleted_at IS NULL").
		Where("supplier_orders.wave_id = ?", q.WaveID)
	var total int64
	if err := query.Distinct("shipments.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := pageOrder(q.SortBy, q.SortDir, map[string]string{
		"shipmentNo":         "shipments.shipment_no",
		"supplierPlatform":   "shipments.supplier_platform",
		"externalShipmentNo": "shipments.external_shipment_no",
		"carrier":            "shipments.carrier_name || shipments.carrier_code",
		"trackingNo":         "shipments.tracking_no",
		"status":             "shipments.status",
		"shippedAt":          "shipments.shipped_at",
	}, "shipments.id")
	var rows []persistence.Shipment
	if err := query.Select("shipments.*").Order(order).Offset(q.Offset).Limit(q.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]domain.Shipment, len(rows))
	for i := range rows {
		items[i] = *persistence.ShipmentToDomain(&rows[i])
	}
	return items, total, nil
}

func (r *listPaginationRepository) ListShipmentLinesByShipmentIDs(ctx context.Context, ids []uint) ([]domain.ShipmentLine, error) {
	if len(ids) == 0 {
		return []domain.ShipmentLine{}, nil
	}
	var rows []persistence.ShipmentLine
	if err := r.db.WithContext(ctx).Where("shipment_id IN ?", ids).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.ShipmentLine, len(rows))
	for i := range rows {
		items[i] = *persistence.ShipmentLineToDomain(&rows[i])
	}
	return items, nil
}
