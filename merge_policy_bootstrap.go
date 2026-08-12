package main

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	"gorm.io/gorm"
)

// migrateLegacyMergePolicy runs once at startup, never from a policy read.
// Presence of the durable policy is the idempotency marker; all eight legacy
// boolean combinations retain their exact raw evidence modes.
func migrateLegacyMergePolicy(ctx context.Context, gdb *gorm.DB) error {
	settings, err := service.NewSettingsService().Load()
	if err != nil {
		return fmt.Errorf("load legacy merge settings: %w", err)
	}
	uc := app.NewMergeGovernanceUseCase(
		infra.NewMergeGovernanceRepository(gdb),
		infra.NewProfileRepository(gdb),
		infra.NewAddressRepository(gdb),
		infra.NewCustomerProfileOriginRepository(gdb),
	)
	_, err = uc.MigrateLegacyPolicy(ctx, app.LegacyMergeSettings{
		AutoMergeCrossPlatform: settings.AutoMergeCrossPlatform,
		AutoMergeByEmail:       settings.AutoMergeByEmail,
		AutoMergeByPhone:       settings.AutoMergeByPhone,
	})
	if err != nil {
		return fmt.Errorf("migrate legacy merge policy: %w", err)
	}
	return nil
}
