package persistence

import (
	"strings"
	"testing"
)

func TestAllocationPolicyRuleToDomain_ValidWaveAll(t *testing.T) {
	t.Parallel()

	rule, err := AllocationPolicyRuleToDomain(&AllocationPolicyRule{
		SelectorPayload: `{"type":"wave_all"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule == nil {
		t.Fatal("expected rule, got nil")
	}
	if rule.SelectorPayload.Type != "wave_all" {
		t.Fatalf("selector type = %q, want wave_all", rule.SelectorPayload.Type)
	}
}

func TestAllocationPolicyRuleToDomain_CorruptJSON(t *testing.T) {
	t.Parallel()

	rule, err := AllocationPolicyRuleToDomain(&AllocationPolicyRule{
		SelectorPayload: `{not-json`,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal selector payload") {
		t.Fatalf("error %q missing unmarshal selector payload prefix", err)
	}
	if rule != nil {
		t.Fatalf("expected nil rule, got %+v", rule)
	}
}

func TestAllocationPolicyRuleToDomain_EmptyPayload(t *testing.T) {
	t.Parallel()

	rule, err := AllocationPolicyRuleToDomain(&AllocationPolicyRule{
		SelectorPayload: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule == nil {
		t.Fatal("expected rule, got nil")
	}
	if rule.SelectorPayload.Type != "" {
		t.Fatalf("selector type = %q, want empty", rule.SelectorPayload.Type)
	}
}
