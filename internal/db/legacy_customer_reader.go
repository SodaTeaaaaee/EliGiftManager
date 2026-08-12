package db

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"
)

type LegacyCustomerSchema struct {
	MembersPresent       bool
	NicknamesPresent     bool
	MemberNicknameColumn string
}

type LegacyMemberRow struct {
	ID          int64
	Platform    string
	PlatformUID string
	Nickname    string
}

type LegacyMemberNicknameRow struct {
	ID           int64
	MemberID     int64
	Nickname     string
	CreatedAtRaw string
}

type LegacyCustomerReader interface {
	Discover(ctx context.Context) (*LegacyCustomerSchema, error)
	ReadMembersAfter(ctx context.Context, afterID int64, limit int, schema *LegacyCustomerSchema) ([]LegacyMemberRow, error)
	ReadNicknamesAfter(ctx context.Context, afterID int64, limit int) ([]LegacyMemberNicknameRow, error)
}

type sqliteLegacyCustomerReader struct{ db *gorm.DB }

type sqliteColumnInfo struct {
	CID     int    `gorm:"column:cid"`
	Name    string `gorm:"column:name"`
	Type    string `gorm:"column:type"`
	NotNull int    `gorm:"column:notnull"`
	Default any    `gorm:"column:dflt_value"`
	PK      int    `gorm:"column:pk"`
}

func newSQLiteLegacyCustomerReader(db *gorm.DB) LegacyCustomerReader {
	return &sqliteLegacyCustomerReader{db: db}
}

func (r *sqliteLegacyCustomerReader) Discover(ctx context.Context) (*LegacyCustomerSchema, error) {
	membersPresent, err := sqliteTableExists(ctx, r.db, "members")
	if err != nil {
		return nil, err
	}
	nicknamesPresent, err := sqliteTableExists(ctx, r.db, "member_nicknames")
	if err != nil {
		return nil, err
	}
	schema := &LegacyCustomerSchema{MembersPresent: membersPresent, NicknamesPresent: nicknamesPresent}
	if !membersPresent {
		if nicknamesPresent {
			return nil, fmt.Errorf("invalid legacy customer schema: member_nicknames exists without members")
		}
		return schema, nil
	}

	members, err := sqliteTableColumns(ctx, r.db, "members")
	if err != nil {
		return nil, err
	}
	if err := validateLegacyColumns("members", members, map[string]string{
		"id": "integer_primary_key", "platform": "text", "platform_uid": "text",
	}); err != nil {
		return nil, err
	}
	for _, candidate := range []string{"nickname", "current_nickname", "display_name", "name"} {
		if _, ok := members[candidate]; ok {
			schema.MemberNicknameColumn = candidate
			break
		}
	}

	if nicknamesPresent {
		nicknames, err := sqliteTableColumns(ctx, r.db, "member_nicknames")
		if err != nil {
			return nil, err
		}
		if err := validateLegacyColumns("member_nicknames", nicknames, map[string]string{
			"id": "integer_primary_key", "member_id": "integer", "nickname": "text", "created_at": "scalar",
		}); err != nil {
			return nil, err
		}
	}
	return schema, nil
}

func (r *sqliteLegacyCustomerReader) ReadMembersAfter(
	ctx context.Context,
	afterID int64,
	limit int,
	schema *LegacyCustomerSchema,
) ([]LegacyMemberRow, error) {
	nicknameExpression := `''`
	if schema != nil && schema.MemberNicknameColumn != "" {
		// Discovery only ever assigns one of the fixed identifiers above.
		nicknameExpression = `COALESCE(CAST(` + schema.MemberNicknameColumn + ` AS TEXT), '')`
	}
	query := `SELECT id, COALESCE(CAST(platform AS TEXT), '') AS platform,
COALESCE(CAST(platform_uid AS TEXT), '') AS platform_uid, ` + nicknameExpression + ` AS nickname
FROM members WHERE id > ? ORDER BY id LIMIT ?`
	var rows []LegacyMemberRow
	if err := r.db.WithContext(ctx).Raw(query, afterID, limit).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("read legacy members after %d: %w", afterID, err)
	}
	return rows, nil
}

func (r *sqliteLegacyCustomerReader) ReadNicknamesAfter(
	ctx context.Context,
	afterID int64,
	limit int,
) ([]LegacyMemberNicknameRow, error) {
	var rows []LegacyMemberNicknameRow
	err := r.db.WithContext(ctx).Raw(`SELECT id, COALESCE(member_id, 0) AS member_id,
COALESCE(CAST(nickname AS TEXT), '') AS nickname,
COALESCE(CAST(created_at AS TEXT), '') AS created_at_raw
FROM member_nicknames WHERE id > ? ORDER BY id LIMIT ?`, afterID, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("read legacy member_nicknames after %d: %w", afterID, err)
	}
	return rows, nil
}

func sqliteTableExists(ctx context.Context, database *gorm.DB, table string) (bool, error) {
	var count int64
	err := database.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table,
	).Scan(&count).Error
	if err != nil {
		return false, fmt.Errorf("discover legacy table %q: %w", table, err)
	}
	return count > 0, nil
}

func sqliteTableColumns(ctx context.Context, database *gorm.DB, table string) (map[string]sqliteColumnInfo, error) {
	var rows []sqliteColumnInfo
	// table is selected from the two fixed legacy table names.
	if err := database.WithContext(ctx).Raw(`PRAGMA table_info(` + table + `)`).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("inspect legacy table %q: %w", table, err)
	}
	columns := make(map[string]sqliteColumnInfo, len(rows))
	for _, row := range rows {
		columns[strings.ToLower(row.Name)] = row
	}
	return columns, nil
}

func validateLegacyColumns(table string, actual map[string]sqliteColumnInfo, required map[string]string) error {
	names := make([]string, 0, len(required))
	for name := range required {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		expected := required[name]
		column, ok := actual[name]
		if !ok {
			return fmt.Errorf("invalid legacy customer schema: %s.%s is required", table, name)
		}
		columnType := strings.ToUpper(strings.TrimSpace(column.Type))
		switch expected {
		case "integer_primary_key":
			if !strings.Contains(columnType, "INT") || column.PK == 0 {
				return fmt.Errorf("invalid legacy customer schema: %s.%s must be an integer primary key", table, name)
			}
		case "integer":
			if !strings.Contains(columnType, "INT") {
				return fmt.Errorf("invalid legacy customer schema: %s.%s must have integer affinity", table, name)
			}
		case "text":
			if !(strings.Contains(columnType, "CHAR") || strings.Contains(columnType, "CLOB") || strings.Contains(columnType, "TEXT")) {
				return fmt.Errorf("invalid legacy customer schema: %s.%s must have text affinity", table, name)
			}
		case "scalar":
			if columnType == "" || strings.Contains(columnType, "BLOB") {
				return fmt.Errorf("invalid legacy customer schema: %s.%s must be a scalar timestamp", table, name)
			}
		}
	}
	return nil
}
