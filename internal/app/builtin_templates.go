package app

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/alignment"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// builtinTemplateSpec is one read-only catalog entry seeded at startup. Exactly
// one built-in row exists per (platform key, document type); its content is
// refreshed on every start so software upgrades propagate.
type builtinTemplateSpec struct {
	platformKey  string
	documentType string
	name         string
	notes        string
	mapping      *alignment.MappingConfig
	layout       *alignment.LayoutConfig
}

func builtinTemplateSpecs() []builtinTemplateSpec {
	membership := builtinMembershipListMapping()
	orders := builtinOrderExportMapping()
	returns := builtinShipmentReturnMapping()
	factory := builtinFactoryOrderLayout()
	writeback := builtinWritebackLayout()
	return []builtinTemplateSpec{
		{
			platformKey: "bilibili", documentType: DocumentTypeMembershipList,
			name:    "哔哩哔哩 会员名单（内置）",
			notes:   "从需求平台导出的会员列表：无表头，三列依次为 等级、UID、昵称。",
			mapping: &membership,
		},
		{
			platformKey: "bilibili", documentType: DocumentTypeOrderExport,
			name:    "哔哩哔哩 订单导出（内置）",
			notes:   "从需求平台导出的订单数据：每行一个订单，商品名称同时作为商品别名 ID，无买家 UID。",
			mapping: &orders,
		},
		{
			platformKey: "rozao", documentType: DocumentTypeShipmentReturn,
			name:    "柔造 快递订单回传（内置）",
			notes:   "从工厂平台导出的快递订单数据，13 列；订单编号即内部追踪标识。",
			mapping: &returns,
		},
		{
			platformKey: "rozao", documentType: DocumentTypeFactoryOrder,
			name:   "柔造 批量下单表格（内置）",
			notes:  "需要导入工厂平台的批量下单表格，六列 xlsx。",
			layout: &factory,
		},
		{
			platformKey: "bilibili", documentType: DocumentTypeWriteback,
			name:   "哔哩哔哩 订单快递跟踪（内置）",
			notes:  "需要导入需求平台的订单快递跟踪表：订单号、快递公司编码（承运商映射的外部 ID）、物流单号。",
			layout: &writeback,
		},
	}
}

// builtinMembershipListMapping is the headerless bilibili membership export:
// 等级, UID, 昵称.
func builtinMembershipListMapping() alignment.MappingConfig {
	return alignment.MappingConfig{
		Version: alignment.MappingSchemaVersion,
		Mode:    alignment.ModePositional,
		Positions: map[string]int{
			"membership.level":      0,
			"identity.value":        1,
			"customer.display_name": 2,
		},
		Transforms: map[string][]string{
			"membership.level":      {"trim"},
			"identity.value":        {"trim"},
			"customer.display_name": {"trim"},
		},
		Required:    []string{"identity.value"},
		Fingerprint: []string{"identity.value"},
	}
}

// builtinOrderExportMapping is the bilibili single-order export (13 header
// columns). The file carries no product id and no buyer uid: the product
// title doubles as the alias id, and the buyer nickname only feeds the
// customer display name.
func builtinOrderExportMapping() alignment.MappingConfig {
	return alignment.MappingConfig{
		Version: alignment.MappingSchemaVersion,
		Mode:    alignment.ModeHeader,
		Columns: map[string]string{
			"source.document_no":      "订单号",
			"product.alias_id":        "商品名称",
			"product.alias_title":     "商品名称",
			"product.alias_spec":      "规格",
			"quantity":                "数量",
			"source.created_at":       "付款时间",
			"customer.display_name":   "买家昵称",
			"recipient.name":          "收货人姓名",
			"recipient.phone":         "联系电话",
			"recipient.address_line1": "收货地址",
		},
		Transforms: map[string][]string{
			"source.document_no":      {"trim", "strip_quotes"},
			"product.alias_id":        {"trim"},
			"product.alias_title":     {"trim"},
			"product.alias_spec":      {"trim"},
			"quantity":                {"trim"},
			"source.created_at":       {"parseDate"},
			"customer.display_name":   {"trim"},
			"recipient.name":          {"trim"},
			"recipient.phone":         {"normalizePhone"},
			"recipient.address_line1": {"trim"},
		},
		Required:    []string{"source.document_no"},
		Fingerprint: []string{"source.document_no"},
	}
}

// builtinShipmentReturnMapping is the rouzao 13-column shipment-return CSV.
func builtinShipmentReturnMapping() alignment.MappingConfig {
	return alignment.MappingConfig{
		Version: alignment.MappingSchemaVersion,
		Mode:    alignment.ModeHeader,
		Columns: map[string]string{
			"tracking.id":             "订单编号",
			"source.created_at":       "下单时间",
			"product.alias_id":        "商品编码",
			"product.alias_title":     "商品名称",
			"product.alias_spec":      "规格&数量",
			"recipient.name":          "收件人",
			"recipient.phone":         "电话",
			"recipient.address_line1": "收件信息",
			"shipment.carrier_name":   "物流公司",
			"shipment.tracking_no":    "物流单号",
			"shipment.shipped_at":     "打印快递时间",
			"shipment.quantity":       "规格&数量",
		},
		Transforms: map[string][]string{
			"tracking.id":           {"trim", "strip_quotes"},
			"shipment.tracking_no":  {"trim", "strip_quotes"},
			"shipment.carrier_name": {"trim"},
			"shipment.shipped_at":   {"parseDate"},
		},
		Required:    []string{"tracking.id", "shipment.tracking_no"},
		Fingerprint: []string{"tracking.id", "shipment.tracking_no"},
	}
}

// builtinFactoryOrderLayout is the rouzao six-column bulk order sheet.
func builtinFactoryOrderLayout() alignment.LayoutConfig {
	return alignment.LayoutConfig{
		Version:     alignment.LayoutSchemaVersion,
		Format:      alignment.FormatXLSX,
		ColumnOrder: []string{"tracking.id", "recipient.name", "recipient.phone", "recipient.address_line1", "product.factory_sku", "quantity"},
		HeaderNames: map[string]string{
			"tracking.id":             "第三方订单号",
			"recipient.name":          "收件人",
			"recipient.phone":         "联系电话",
			"recipient.address_line1": "收件地址",
			"product.factory_sku":     "商家编码",
			"quantity":                "下单数量",
		},
	}
}

// builtinWritebackLayout is the bilibili order tracking import sheet: order
// number, the carrier code bilibili publishes, and the tracking number. Header
// names mirror bilibili's own template, asterisks included.
func builtinWritebackLayout() alignment.LayoutConfig {
	return alignment.LayoutConfig{
		Version:     alignment.LayoutSchemaVersion,
		Format:      alignment.FormatXLSX,
		ColumnOrder: []string{"source.document_no", "shipment.carrier_code", "shipment.tracking_no"},
		HeaderNames: map[string]string{
			"source.document_no":    "订单号*",
			"shipment.carrier_code": "快递公司编码*（请在网页中查看快递编码）",
			"shipment.tracking_no":  "物流单号*",
		},
	}
}

// EnsureBuiltinTemplates upserts the read-only template catalog: one Builtin
// row per (platform, document type, direction). Existing rows keep their id;
// when the seeded content differs the row is overwritten and its version
// advances, so upgrades propagate while unchanged startups write nothing.
// Platforms must already exist (EnsureBuiltinPlatforms runs first).
func (ws *Workspace) EnsureBuiltinTemplates(ctx context.Context) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		existing, err := tx.ListTemplates(ctx)
		if err != nil {
			return err
		}
		for _, spec := range builtinTemplateSpecs() {
			platform, err := tx.GetPlatformByKey(ctx, spec.platformKey)
			if err != nil {
				return fmt.Errorf("seed builtin templates: platform %q: %w", spec.platformKey, err)
			}
			info, ok := documentTypeInfo(spec.documentType)
			if !ok {
				return fmt.Errorf("seed builtin templates: unknown document type %q", spec.documentType)
			}
			want := domain.TemplateConfig{
				PlatformID:   platform.ID,
				DocumentType: spec.documentType,
				Direction:    info.Direction,
				Name:         spec.name,
				Notes:        spec.notes,
				Builtin:      true,
			}
			if spec.mapping != nil {
				raw, err := alignment.SerializeMappingConfig(*spec.mapping)
				if err != nil {
					return err
				}
				want.MappingJSON = raw
			}
			if spec.layout != nil {
				raw, err := alignment.SerializeLayoutConfig(*spec.layout)
				if err != nil {
					return err
				}
				want.LayoutJSON = raw
			}
			if err := validateTemplate(ctx, tx, &want); err != nil {
				return fmt.Errorf("seed builtin templates: %s: %w", spec.name, err)
			}
			var current *domain.TemplateConfig
			for i := range existing {
				e := &existing[i]
				if e.Builtin && e.PlatformID == want.PlatformID && e.DocumentType == want.DocumentType && e.Direction == want.Direction {
					current = e
					break
				}
			}
			if current == nil {
				want.Version = 1
				if err := tx.CreateTemplate(ctx, &want); err != nil {
					return err
				}
				continue
			}
			if current.Name == want.Name && current.Notes == want.Notes && current.MappingJSON == want.MappingJSON && current.LayoutJSON == want.LayoutJSON {
				continue
			}
			next := *current
			next.Name, next.Notes = want.Name, want.Notes
			next.MappingJSON, next.LayoutJSON = want.MappingJSON, want.LayoutJSON
			next.Version = current.Version + 1
			if err := tx.UpdateTemplate(ctx, &next); err != nil {
				return err
			}
		}
		return nil
	})
}
