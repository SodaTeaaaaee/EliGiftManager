package app

import (
	"context"
	"strings"
	"testing"
)

// These tests must not call t.Parallel(): Parallel tests are parked until every
// sequential test in the package finishes. A slow or hung sequential test
// (catalog zip, sampledata, sqlite) then makes this file look like it hung for
// the full go-test timeout even though Resolve returns immediately.

func TestDemandCustomerResolution_StableStrategyRequiresSourceCustomerRef(t *testing.T) {
	svc := NewDemandCustomerResolutionService(nil, nil)

	_, err := svc.Resolve(context.Background(), DemandCustomerResolutionInput{
		IdentityStrategy:  "platform_uid",
		SourceChannel:     "bilibili",
		SourceCustomerRef: "",
	})
	if err == nil {
		t.Fatal("expected error for empty sourceCustomerRef, got nil")
	}
	if !strings.Contains(err.Error(), "sourceCustomerRef") || !strings.Contains(err.Error(), "sourceChannel") {
		t.Errorf("error %q should require sourceCustomerRef and sourceChannel", err.Error())
	}
}

func TestDemandCustomerResolution_StableStrategyRequiresSourceChannel(t *testing.T) {
	svc := NewDemandCustomerResolutionService(nil, nil)

	_, err := svc.Resolve(context.Background(), DemandCustomerResolutionInput{
		IdentityStrategy:  "platform_uid",
		SourceChannel:     "",
		SourceCustomerRef: "uid-1",
	})
	if err == nil {
		t.Fatal("expected error for empty sourceChannel, got nil")
	}
	if !strings.Contains(err.Error(), "sourceCustomerRef") || !strings.Contains(err.Error(), "sourceChannel") {
		t.Errorf("error %q should require sourceCustomerRef and sourceChannel", err.Error())
	}
}

func TestDemandCustomerResolution_RejectsWhitespaceSourceCustomerRef(t *testing.T) {
	svc := NewDemandCustomerResolutionService(nil, nil)

	_, err := svc.Resolve(context.Background(), DemandCustomerResolutionInput{
		IdentityStrategy:  "",
		SourceChannel:     "bilibili",
		SourceCustomerRef: "   ",
	})
	if err == nil {
		t.Fatal("expected error for whitespace sourceCustomerRef, got nil")
	}
}
