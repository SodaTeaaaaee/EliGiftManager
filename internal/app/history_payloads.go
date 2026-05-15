package app

import (
	"encoding/json"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func BuildRuleRestorePatch(op string, rule *domain.AllocationPolicyRule) (string, error) {
	data, err := json.Marshal(rule)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"op":%q,"rule_id":%d,"wave_id":%d,"data":%s}`, op, rule.ID, rule.WaveID, data), nil
}

func BuildRuleUpdatePatch(rule *domain.AllocationPolicyRule) (string, error) {
	data, err := json.Marshal(rule)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"op":"update_rule","rule_id":%d,"wave_id":%d,"data":%s}`, rule.ID, rule.WaveID, data), nil
}

func BuildAdjustmentPatch(op string, adj *domain.FulfillmentAdjustment) (string, error) {
	data, err := json.Marshal(adj)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"op":%q,"adjustment_id":%d,"wave_id":%d,"data":%s}`, op, adj.ID, adj.WaveID, data), nil
}

