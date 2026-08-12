package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openIdentityResolutionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "identity.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&persistence.CustomerProfile{}, &persistence.CustomerIdentity{}); err != nil {
		t.Fatal(err)
	}
	seedEnabledFeaturePolicyForFocusedDB(t, db)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func TestResolveStableProfileConcurrentSameUIDReusesOneProfile(t *testing.T) {
	db := openIdentityResolutionTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	input := StableIdentityResolutionInput{
		Namespace: "bilibili", IdentityPlatform: "BILIBILI", IdentityValue: "uid-1", IdentityType: "platform_uid",
		ObservedAt: time.Now().UTC(), InitialDisplayName: "Alice", ProfileType: "member",
	}
	results := make([]uint, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = db.Transaction(func(tx *gorm.DB) error {
				result, err := NewIdentityResolutionService(infra.NewProfileRepository(tx)).ResolveStableProfile(context.Background(), input)
				if err == nil {
					results[i] = result.CustomerProfileID
				}
				return err
			})
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if results[0] == 0 || results[0] != results[1] {
		t.Fatalf("results=%v", results)
	}
	var profileCount, identityCount int64
	_ = db.Model(&persistence.CustomerProfile{}).Count(&profileCount).Error
	_ = db.Model(&persistence.CustomerIdentity{}).Count(&identityCount).Error
	if profileCount != 1 || identityCount != 1 {
		t.Fatalf("profiles=%d identities=%d", profileCount, identityCount)
	}
}

func TestResolveStableProfileRejectsAmbiguousLegacyDuplicates(t *testing.T) {
	db := openIdentityResolutionTestDB(t)
	repo := infra.NewProfileRepository(db)
	for i := 0; i < 2; i++ {
		profile := &domain.CustomerProfile{DisplayName: fmt.Sprintf("p%d", i), ProfileType: "member", Status: domain.CustomerProfileStatusActive, RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
		if err := repo.Create(context.Background(), profile); err != nil {
			t.Fatal(err)
		}
		identity := &domain.CustomerIdentity{CustomerProfileID: profile.ID, IdentityPlatform: "bilibili", IdentityValue: "uid-dup", IdentityType: "platform_uid", Namespace: "bilibili", NormalizedValue: "uid-dup"}
		if err := repo.CreateIdentity(context.Background(), identity); err != nil {
			t.Fatal(err)
		}
	}
	_, err := NewIdentityResolutionService(repo).ResolveStableProfile(context.Background(), StableIdentityResolutionInput{
		Namespace: "bilibili", IdentityPlatform: "bilibili", IdentityValue: "uid-dup", IdentityType: "platform_uid",
	})
	var ambiguous *AmbiguousIdentityError
	if !errors.As(err, &ambiguous) || ambiguous.Count != 2 {
		t.Fatalf("err=%v, want AmbiguousIdentityError count=2", err)
	}
}

func TestResolveStableProfileRollsBackProfileAndIdentityTogether(t *testing.T) {
	db := openIdentityResolutionTestDB(t)
	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := NewIdentityResolutionService(infra.NewProfileRepository(tx)).ResolveStableProfile(context.Background(), StableIdentityResolutionInput{
			Namespace: "bilibili", IdentityPlatform: "bilibili", IdentityValue: "uid-rollback", IdentityType: "platform_uid",
		})
		if err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}
	var profileCount, identityCount int64
	_ = db.Model(&persistence.CustomerProfile{}).Count(&profileCount).Error
	_ = db.Model(&persistence.CustomerIdentity{}).Count(&identityCount).Error
	if profileCount != 0 || identityCount != 0 {
		t.Fatalf("profiles=%d identities=%d", profileCount, identityCount)
	}
}
