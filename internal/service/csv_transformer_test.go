package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestParseMemberCSVSuccess(t *testing.T) {
	t.Parallel()

	csvFile := writeTestCSVFile(t, []string{
		"平台名,用户ID,昵称,姓名,手机号,详细地址,会员等级,积分",
		"抖音,uid-001,小李,李雷,13800000000,上海市浦东新区,VIP,128",
	})

	template := model.TemplateConfig{
		Type: model.TemplateTypeImportMember,
		Name: "会员导入模板",
		MappingRules: `{
			"platform": "平台名",
			"platform_uid": "用户ID",
			"nickname": "昵称"
		}`,
	}

	members, err := ParseMemberCSV(csvFile, template)
	if err != nil {
		t.Fatalf("ParseMemberCSV returned unexpected error: %v", err)
	}

	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}

	member := members[0]
	if member.Platform != "抖音" {
		t.Fatalf("expected Platform to be 抖音, got %q", member.Platform)
	}
	if member.PlatformUID != "uid-001" {
		t.Fatalf("expected PlatformUID to be uid-001, got %q", member.PlatformUID)
	}

	var extraData map[string]string
	if err := json.Unmarshal([]byte(member.ExtraData), &extraData); err != nil {
		t.Fatalf("failed to unmarshal ExtraData: %v", err)
	}

	if extraData["姓名"] != "李雷" {
		t.Fatalf("expected ExtraData[姓名] to be 李雷, got %q", extraData["姓名"])
	}
	if extraData["手机号"] != "13800000000" {
		t.Fatalf("expected ExtraData[手机号] to be 13800000000, got %q", extraData["手机号"])
	}
	if extraData["详细地址"] != "上海市浦东新区" {
		t.Fatalf("expected ExtraData[详细地址] to be 上海市浦东新区, got %q", extraData["详细地址"])
	}
	if extraData["会员等级"] != "VIP" {
		t.Fatalf("expected ExtraData[会员等级] to be VIP, got %q", extraData["会员等级"])
	}
	if extraData["积分"] != "128" {
		t.Fatalf("expected ExtraData[积分] to be 128, got %q", extraData["积分"])
	}
}

func TestParseMemberCSVSupportsChineseInternalFieldAliases(t *testing.T) {
	t.Parallel()

	csvFile := writeTestCSVFile(t, []string{
		"平台,平台UID,昵称",
		"快手,ks-1001,阿星",
	})

	template := model.TemplateConfig{
		Type: model.TemplateTypeImportMember,
		Name: "中文字段模板",
		MappingRules: `{
			"平台名称": "平台",
			"平台用户ID": "平台UID",
			"昵称": "昵称"
		}`,
	}

	members, err := ParseMemberCSV(csvFile, template)
	if err != nil {
		t.Fatalf("ParseMemberCSV returned unexpected error: %v", err)
	}

	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}

	member := members[0]
	if member.Platform != "快手" || member.PlatformUID != "ks-1001" {
		t.Fatalf("unexpected member content: %+v", member)
	}
	if member.ExtraData != "{}" {
		t.Fatalf("expected empty ExtraData JSON, got %q", member.ExtraData)
	}
}

func TestParseMemberCSVRejectsLegacyAddressFieldMapping(t *testing.T) {
	t.Parallel()

	csvFile := writeTestCSVFile(t, []string{
		"平台,平台UID,昵称,收货地址",
		"淘宝,tb-001,小王,杭州市余杭区",
	})

	template := model.TemplateConfig{
		Type: model.TemplateTypeImportMember,
		Name: "旧模板",
		MappingRules: `{
			"platform": "平台",
			"platform_uid": "平台UID",
			"nickname": "昵称",
			"address": "收货地址"
		}`,
	}

	_, err := ParseMemberCSV(csvFile, template)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported member standard field") {
		t.Fatalf("expected unsupported field error, got %v", err)
	}
}

func TestParseMemberCSVRejectsInvalidTemplateType(t *testing.T) {
	t.Parallel()

	csvFile := writeTestCSVFile(t, []string{
		"平台,平台UID,昵称",
		"快手,ks-1001,阿星",
	})

	template := model.TemplateConfig{
		Type:         model.TemplateTypeExportOrder,
		Name:         "错误模板",
		MappingRules: `{"platform":"平台","platform_uid":"平台UID","nickname":"昵称"}`,
	}

	_, err := ParseMemberCSV(csvFile, template)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "not applicable to member import") {
		t.Fatalf("expected invalid template type error, got %v", err)
	}
}

func TestParseMemberCSVMissingRequiredField(t *testing.T) {
	t.Parallel()

	csvFile := writeTestCSVFile(t, []string{
		"平台,平台UID,昵称",
		"快手,ks-1001,",
	})

	template := model.TemplateConfig{
		Type:         model.TemplateTypeImportMember,
		Name:         "缺少昵称模板",
		MappingRules: `{"platform":"平台","platform_uid":"平台UID","nickname":"昵称"}`,
	}

	_, err := ParseMemberCSV(csvFile, template)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "missing required field Nickname") {
		t.Fatalf("expected missing nickname error, got %v", err)
	}
}

func TestParseMemberCSVRejectsMissingMappedHeader(t *testing.T) {
	t.Parallel()

	csvFile := writeTestCSVFile(t, []string{
		"平台,昵称",
		"淘宝,小王",
	})

	template := model.TemplateConfig{
		Type:         model.TemplateTypeImportMember,
		Name:         "缺表头模板",
		MappingRules: `{"platform":"平台","platform_uid":"平台UID","nickname":"昵称"}`,
	}

	_, err := ParseMemberCSV(csvFile, template)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "external header") {
		t.Fatalf("expected missing header error, got %v", err)
	}
}
