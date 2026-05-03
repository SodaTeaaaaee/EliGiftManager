package service

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

func upsertMember(db *gorm.DB, candidate model.Member) (model.Member, error) {
	var existing model.Member
	err := db.Where("platform = ? AND platform_uid = ?", candidate.Platform, candidate.PlatformUID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if createErr := db.Create(&candidate).Error; createErr != nil {
			return model.Member{}, fmt.Errorf("create member failed: %w", createErr)
		}

		return candidate, nil
	}
	if err != nil {
		return model.Member{}, fmt.Errorf("query member failed: %w", err)
	}

	existing.ExtraData = candidate.ExtraData
	if saveErr := db.Save(&existing).Error; saveErr != nil {
		return model.Member{}, fmt.Errorf("update member failed: %w", saveErr)
	}

	return existing, nil
}

func ensureLatestNickname(db *gorm.DB, memberID uint, nickname string) error {
	if strings.TrimSpace(nickname) == "" {
		return fmt.Errorf("nickname is required")
	}

	latestNickname, err := getLatestMemberNickname(db, memberID)
	if err != nil {
		return fmt.Errorf("query latest nickname failed: %w", err)
	}

	if latestNickname != nil && latestNickname.Nickname == nickname {
		return nil
	}

	record := model.MemberNickname{
		MemberID: memberID,
		Nickname: nickname,
	}

	if err := db.Create(&record).Error; err != nil {
		return fmt.Errorf("create nickname history record failed: %w", err)
	}

	return nil
}

func ensureTemplateType(template model.TemplateConfig, expectedType string, scene string) error {
	if template.Type != "" && template.Type != expectedType {
		return fmt.Errorf("template type %q is not applicable to %s", template.Type, scene)
	}

	return nil
}

func stripBOM(r io.Reader) io.Reader {
	br := bufio.NewReader(r)
	bom := []byte{0xEF, 0xBB, 0xBF}
	peek, _ := br.Peek(3)
	if len(peek) >= 3 && peek[0] == bom[0] && peek[1] == bom[1] && peek[2] == bom[2] {
		_, _ = br.Discard(3)
	}
	return br
}

func readCSVCell(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}

	return strings.TrimSpace(record[index])
}

func isEmptyRecord(record []string) bool {
	for _, item := range record {
		if strings.TrimSpace(item) != "" {
			return false
		}
	}

	return true
}
