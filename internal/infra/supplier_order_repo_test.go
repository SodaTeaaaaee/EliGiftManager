package infra

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeleteDraftsByWaveAndFactoryProfilePreservesOtherFactory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scoped_factory_drafts?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&persistence.SupplierOrder{}, &persistence.SupplierOrderLine{}); err != nil {
		t.Fatal(err)
	}
	repo := NewSupplierOrderRepository(db)
	scoped, ok := repo.(interface {
		DeleteDraftsByWaveAndFactoryProfile(context.Context, uint, uint) error
	})
	if !ok {
		t.Fatal("supplier repository does not support factory-scoped draft rebuild")
	}

	profileA, profileB := uint(10), uint(20)
	orderA := &domain.SupplierOrder{WaveID: 1, FactoryIntegrationProfileID: &profileA, Status: "draft", SubmissionMode: "csv"}
	orderB := &domain.SupplierOrder{WaveID: 1, FactoryIntegrationProfileID: &profileB, Status: "draft", SubmissionMode: "csv"}
	for _, order := range []*domain.SupplierOrder{orderA, orderB} {
		if err := repo.AtomicCreateSupplierOrder(context.Background(), order, []*domain.SupplierOrderLine{{FulfillmentLineID: order.WaveID, SubmittedQuantity: 1}}, nil); err != nil {
			t.Fatalf("create order: %v", err)
		}
	}
	if err := scoped.DeleteDraftsByWaveAndFactoryProfile(context.Background(), 1, profileA); err != nil {
		t.Fatalf("DeleteDraftsByWaveAndFactoryProfile: %v", err)
	}
	orders, err := repo.ListByWave(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 1 || orders[0].FactoryIntegrationProfileID == nil || *orders[0].FactoryIntegrationProfileID != profileB {
		t.Fatalf("remaining orders = %+v", orders)
	}
	linesA, err := repo.ListLinesByOrder(context.Background(), orderA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(linesA) != 0 {
		t.Fatalf("deleted factory lines remain: %+v", linesA)
	}
}
