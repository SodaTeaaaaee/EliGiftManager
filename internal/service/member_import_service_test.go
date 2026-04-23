package service

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestImportMembersFromCSVTracksNicknameHistory(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)
	template := model.TemplateConfig{
		Type: model.TemplateTypeImportMember,
		Name: "会员导入模板",
		MappingRules: `{
			"platform": "平台",
			"platform_uid": "平台UID",
			"nickname": "昵称"
		}`,
	}

	firstCSV := writeTestCSVFile(t, []string{
		"平台,平台UID,昵称,等级",
		"抖音,uid-001,旧昵称,VIP",
	})
	importedMembers, err := ImportMembersFromCSV(db, firstCSV, template)
	if err != nil {
		t.Fatalf("ImportMembersFromCSV returned unexpected error: %v", err)
	}

	if len(importedMembers) != 1 {
		t.Fatalf("expected 1 imported member, got %d", len(importedMembers))
	}
	if importedMembers[0].ID == 0 {
		t.Fatal("expected imported member to have a database id")
	}

	var nicknameCount int64
	if err := db.Model(&model.MemberNickname{}).Count(&nicknameCount).Error; err != nil {
		t.Fatalf("failed to count nickname history: %v", err)
	}
	if nicknameCount != 1 {
		t.Fatalf("expected 1 nickname history record after first import, got %d", nicknameCount)
	}

	secondCSV := writeTestCSVFile(t, []string{
		"平台,平台UID,昵称,等级",
		"抖音,uid-001,旧昵称,SVIP",
	})
	if _, err := ImportMembersFromCSV(db, secondCSV, template); err != nil {
		t.Fatalf("second ImportMembersFromCSV returned unexpected error: %v", err)
	}

	if err := db.Model(&model.MemberNickname{}).Count(&nicknameCount).Error; err != nil {
		t.Fatalf("failed to count nickname history after second import: %v", err)
	}
	if nicknameCount != 1 {
		t.Fatalf("expected nickname history count to remain 1, got %d", nicknameCount)
	}

	thirdCSV := writeTestCSVFile(t, []string{
		"平台,平台UID,昵称,等级",
		"抖音,uid-001,新昵称,SVIP",
	})
	if _, err := ImportMembersFromCSV(db, thirdCSV, template); err != nil {
		t.Fatalf("third ImportMembersFromCSV returned unexpected error: %v", err)
	}

	if err := db.Model(&model.MemberNickname{}).Count(&nicknameCount).Error; err != nil {
		t.Fatalf("failed to count nickname history after third import: %v", err)
	}
	if nicknameCount != 2 {
		t.Fatalf("expected nickname history count to become 2, got %d", nicknameCount)
	}

	var latestNickname model.MemberNickname
	if err := db.Order("created_at DESC").Order("id DESC").First(&latestNickname).Error; err != nil {
		t.Fatalf("failed to query latest nickname: %v", err)
	}
	if latestNickname.Nickname != "新昵称" {
		t.Fatalf("expected latest nickname to be 新昵称, got %q", latestNickname.Nickname)
	}

	var addressCount int64
	if err := db.Model(&model.MemberAddress{}).Count(&addressCount).Error; err != nil {
		t.Fatalf("failed to count member addresses: %v", err)
	}
	if addressCount != 0 {
		t.Fatalf("expected no member addresses to be created during import, got %d", addressCount)
	}
}
