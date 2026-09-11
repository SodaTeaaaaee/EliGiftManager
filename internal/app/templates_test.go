package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/alignment"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestDocumentTypeCatalog_LocksDirectionAndPlatformKind(t *testing.T) {
	ws := newTestWorkspace(t)
	catalog := ws.DocumentTypeCatalog()
	want := map[string][2]string{
		DocumentTypeMembershipList: {"input", "source"},
		DocumentTypeOrderExport:    {"input", "source"},
		DocumentTypeShipmentReturn: {"input", "factory"},
		DocumentTypeFactoryOrder:   {"output", "factory"},
		DocumentTypeWriteback:      {"output", "source"},
	}
	if len(catalog) != len(want) {
		t.Fatalf("catalog = %+v, want %d rows", catalog, len(want))
	}
	for _, info := range catalog {
		w, ok := want[info.Key]
		if !ok {
			t.Fatalf("unexpected document type %q", info.Key)
		}
		if info.Direction != w[0] || info.PlatformKind != w[1] {
			t.Fatalf("%s = %s/%s, want %s/%s", info.Key, info.Direction, info.PlatformKind, w[0], w[1])
		}
	}
	// The returned slice is a copy: mutating it does not touch the catalog.
	catalog[0].Key = "mutated"
	if ws.DocumentTypeCatalog()[0].Key == "mutated" {
		t.Fatal("DocumentTypeCatalog must return a copy")
	}
}

func TestEnsureBuiltinTemplates_IdempotentAndUpgradeable(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	seedBuiltins(t, ws)
	source := platformByKind(t, ws, domain.PlatformKindSource)
	factory := platformByKind(t, ws, domain.PlatformKindFactory)

	first, err := ws.ListTemplates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 5 {
		t.Fatalf("seeded templates = %d, want 5", len(first))
	}
	type key struct {
		platform uint
		docType  string
	}
	seen := map[key]domain.TemplateConfig{}
	for _, tpl := range first {
		if !tpl.Builtin || tpl.Version != 1 {
			t.Fatalf("seeded row must be builtin v1: %+v", tpl)
		}
		info, ok := documentTypeInfo(tpl.DocumentType)
		if !ok || tpl.Direction != info.Direction {
			t.Fatalf("seeded row direction = %q for %s", tpl.Direction, tpl.DocumentType)
		}
		seen[key{tpl.PlatformID, tpl.DocumentType}] = tpl
	}
	for _, want := range []key{
		{source.ID, DocumentTypeMembershipList},
		{source.ID, DocumentTypeOrderExport},
		{source.ID, DocumentTypeWriteback},
		{factory.ID, DocumentTypeShipmentReturn},
		{factory.ID, DocumentTypeFactoryOrder},
	} {
		if _, ok := seen[want]; !ok {
			t.Fatalf("missing builtin %s on platform %d", want.docType, want.platform)
		}
	}
	membership := seen[key{source.ID, DocumentTypeMembershipList}]
	if membership.Name != "哔哩哔哩 会员名单（内置）" {
		t.Fatalf("membership builtin name = %q", membership.Name)
	}
	mapping, err := alignment.ParseMappingConfig(membership.MappingJSON)
	if err != nil {
		t.Fatal(err)
	}
	if mapping.Mode != alignment.ModePositional || mapping.Positions["customer.display_name"] != 2 || mapping.Positions["identity.value"] != 1 || mapping.Positions["membership.level"] != 0 {
		t.Fatalf("membership mapping = %+v", mapping)
	}
	orders := seen[key{source.ID, DocumentTypeOrderExport}]
	orderMapping, err := alignment.ParseMappingConfig(orders.MappingJSON)
	if err != nil {
		t.Fatal(err)
	}
	if orderMapping.Columns["source.document_no"] != "订单号" || orderMapping.Columns["recipient.address_line1"] != "收货地址" || orderMapping.Columns["customer.display_name"] != "买家昵称" {
		t.Fatalf("order export mapping = %+v", orderMapping.Columns)
	}
	writeback := seen[key{source.ID, DocumentTypeWriteback}]
	layout, err := alignment.ParseLayoutConfig(writeback.LayoutJSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(layout.ColumnOrder) != 3 || layout.ColumnOrder[1] != "shipment.carrier_code" || layout.ColumnOrder[2] != "shipment.tracking_no" {
		t.Fatalf("writeback layout = %+v", layout)
	}

	// Seeding again is a no-op: same ids, same versions, same timestamps.
	if err := ws.EnsureBuiltinTemplates(ctx); err != nil {
		t.Fatal(err)
	}
	second, err := ws.ListTemplates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 5 {
		t.Fatalf("templates after re-seed = %d, want 5", len(second))
	}
	for i := range second {
		if second[i].ID != first[i].ID || second[i].Version != first[i].Version || !second[i].UpdatedAt.Equal(first[i].UpdatedAt) {
			t.Fatalf("re-seed changed row %d: %+v vs %+v", i, second[i], first[i])
		}
	}

	// A shipped content change (simulated by an out-of-date row) is
	// overwritten in place with the version advanced, keeping the id.
	stale := membership
	stale.MappingJSON = mustSerializeMapping(t, alignment.MappingConfig{Mode: alignment.ModePositional, Positions: map[string]int{"identity.value": 0}})
	if err := ws.Store.UpdateTemplate(ctx, &stale); err != nil {
		t.Fatal(err)
	}
	if err := ws.EnsureBuiltinTemplates(ctx); err != nil {
		t.Fatal(err)
	}
	refreshed, err := ws.GetTemplate(ctx, membership.ID)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Version != 2 || refreshed.MappingJSON != membership.MappingJSON || !refreshed.Builtin {
		t.Fatalf("refreshed builtin = %+v, want v2 with seeded content", refreshed)
	}
	// Builtins do not count as active templates for any operation.
	if tpl, err := findActiveTemplate(ctx, ws.Store, source.ID, domain.TemplateDirectionInput, DocumentTypeMembershipList); err != nil || tpl != nil {
		t.Fatalf("findActiveTemplate = %+v, %v; want nil for builtin-only", tpl, err)
	}
}

func TestCreateTemplate_ValidatesClosedSetDirectionKindAndConfig(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	seedBuiltins(t, ws)
	source := platformByKind(t, ws, domain.PlatformKindSource)
	factory := platformByKind(t, ws, domain.PlatformKindFactory)
	goodMapping := mustSerializeMapping(t, alignment.MappingConfig{Mode: alignment.ModePositional, Positions: map[string]int{"identity.value": 0}})
	goodLayout, err := alignment.SerializeLayoutConfig(alignment.LayoutConfig{Format: alignment.FormatCSV, ColumnOrder: []string{"source.document_no"}})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		tpl  domain.TemplateConfig
	}{
		{"unknown document type", domain.TemplateConfig{PlatformID: source.ID, DocumentType: "invoice", Direction: "input", Name: "x", MappingJSON: goodMapping}},
		{"wrong direction", domain.TemplateConfig{PlatformID: source.ID, DocumentType: DocumentTypeMembershipList, Direction: "output", Name: "x", MappingJSON: goodMapping}},
		{"wrong platform kind", domain.TemplateConfig{PlatformID: factory.ID, DocumentType: DocumentTypeMembershipList, Direction: "input", Name: "x", MappingJSON: goodMapping}},
		{"writeback on factory", domain.TemplateConfig{PlatformID: factory.ID, DocumentType: DocumentTypeWriteback, Direction: "output", Name: "x", LayoutJSON: goodLayout}},
		{"empty mapping", domain.TemplateConfig{PlatformID: source.ID, DocumentType: DocumentTypeOrderExport, Direction: "input", Name: "x", MappingJSON: ""}},
		{"broken mapping json", domain.TemplateConfig{PlatformID: source.ID, DocumentType: DocumentTypeOrderExport, Direction: "input", Name: "x", MappingJSON: "{"}},
		{"mapping without columns", domain.TemplateConfig{PlatformID: source.ID, DocumentType: DocumentTypeOrderExport, Direction: "input", Name: "x", MappingJSON: `{"version":3,"mode":"header"}`}},
		{"empty layout", domain.TemplateConfig{PlatformID: factory.ID, DocumentType: DocumentTypeFactoryOrder, Direction: "output", Name: "x", LayoutJSON: ""}},
		{"layout without columns", domain.TemplateConfig{PlatformID: factory.ID, DocumentType: DocumentTypeFactoryOrder, Direction: "output", Name: "x", LayoutJSON: `{"version":1,"format":"csv","columnOrder":[]}`}},
		{"missing name", domain.TemplateConfig{PlatformID: source.ID, DocumentType: DocumentTypeMembershipList, Direction: "input", Name: "  ", MappingJSON: goodMapping}},
		{"missing platform", domain.TemplateConfig{PlatformID: 9999, DocumentType: DocumentTypeMembershipList, Direction: "input", Name: "x", MappingJSON: goodMapping}},
	}
	for _, c := range cases {
		tpl := c.tpl
		err := ws.CreateTemplate(ctx, &tpl)
		if !errors.Is(err, ErrInvalidTemplate) {
			t.Fatalf("%s: err = %v, want ErrInvalidTemplate", c.name, err)
		}
	}

	// Valid input template: Builtin and Version are forced regardless of input.
	tpl := &domain.TemplateConfig{PlatformID: source.ID, DocumentType: DocumentTypeMembershipList, Direction: "input", Name: "ok", Version: 9, Builtin: true, MappingJSON: goodMapping}
	if err := ws.CreateTemplate(ctx, tpl); err != nil {
		t.Fatalf("valid create: %v", err)
	}
	if tpl.ID == 0 || tpl.Builtin || tpl.Version != 1 {
		t.Fatalf("created = %+v, want active v1", tpl)
	}
	// Valid output template with a layout only.
	out := &domain.TemplateConfig{PlatformID: source.ID, DocumentType: DocumentTypeWriteback, Direction: "output", Name: "wb", LayoutJSON: goodLayout}
	if err := ws.CreateTemplate(ctx, out); err != nil {
		t.Fatalf("valid output create: %v", err)
	}
	// Six builtins + two user templates are listed together.
	all, err := ws.ListTemplates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	builtins, users := 0, 0
	for _, x := range all {
		if x.Builtin {
			builtins++
		} else {
			users++
		}
	}
	if builtins != 5 || users != 2 {
		t.Fatalf("listed builtins/users = %d/%d, want 5/2", builtins, users)
	}
}

func TestUpdateTemplate_InPlaceVersionBumpAndRefusals(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	seedBuiltins(t, ws)
	source := platformByKind(t, ws, domain.PlatformKindSource)
	tpl := cloneBuiltinTemplate(t, ws, source.ID, DocumentTypeMembershipList)

	edited := *tpl
	edited.Name = "renamed"
	edited.Notes = "tuned"
	edited.MappingJSON = mustSerializeMapping(t, alignment.MappingConfig{Mode: alignment.ModePositional, Positions: map[string]int{"identity.value": 1}})
	updated, err := ws.UpdateTemplate(ctx, &edited)
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	if updated.ID != tpl.ID || updated.Version != 2 || updated.Name != "renamed" || updated.Notes != "tuned" || updated.MappingJSON != edited.MappingJSON {
		t.Fatalf("updated = %+v, want same id, v2, new content", updated)
	}
	stored, err := ws.GetTemplate(ctx, tpl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Version != 2 || stored.Name != "renamed" || stored.Builtin {
		t.Fatalf("stored = %+v", stored)
	}
	if !stored.CreatedAt.Equal(tpl.CreatedAt) {
		t.Fatalf("created_at changed: %v -> %v", tpl.CreatedAt, stored.CreatedAt)
	}
	all, err := ws.ListTemplates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 6 {
		t.Fatalf("templates = %d, want 5 builtins + 1 user (no new row on edit)", len(all))
	}

	// Invalid content is refused and leaves the row untouched.
	bad := *stored
	bad.MappingJSON = ""
	if _, err := ws.UpdateTemplate(ctx, &bad); !errors.Is(err, ErrInvalidTemplate) {
		t.Fatalf("invalid update err = %v, want ErrInvalidTemplate", err)
	}
	after, err := ws.GetTemplate(ctx, tpl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Version != 2 || after.MappingJSON != edited.MappingJSON {
		t.Fatalf("failed update must not touch the row: %+v", after)
	}

	// Platform, document type, and direction are fixed after creation.
	moved := *stored
	moved.DocumentType = DocumentTypeOrderExport
	if _, err := ws.UpdateTemplate(ctx, &moved); !errors.Is(err, ErrInvalidTemplate) {
		t.Fatalf("document type change err = %v, want ErrInvalidTemplate", err)
	}

	// Built-in rows cannot be updated or deleted.
	builtin := builtinTemplate(t, ws, source.ID, DocumentTypeMembershipList)
	builtin.Name = "hacked"
	if _, err := ws.UpdateTemplate(ctx, &builtin); !errors.Is(err, ErrBuiltinTemplate) {
		t.Fatalf("builtin update err = %v, want ErrBuiltinTemplate", err)
	}
	if err := ws.DeleteTemplate(ctx, builtin.ID); !errors.Is(err, ErrBuiltinTemplate) {
		t.Fatalf("builtin delete err = %v, want ErrBuiltinTemplate", err)
	}
	if _, err := ws.UpdateTemplate(ctx, &domain.TemplateConfig{ID: 99999}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing update err = %v, want ErrNotFound", err)
	}
	if err := ws.DeleteTemplate(ctx, 99999); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing delete err = %v, want ErrNotFound", err)
	}
}

func TestDeleteTemplate_LeavesSnapshotsIntact(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	seedBuiltins(t, ws)
	source := platformByKind(t, ws, domain.PlatformKindSource)
	tpl := cloneBuiltinTemplate(t, ws, source.ID, DocumentTypeMembershipList)

	path := filepath.Join("..", "..", "testdata", "integration_profile", "bilibili_membership_positional.csv")
	result, err := ws.ImportFile(ctx, source.ID, tpl.ID, path)
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}
	if err := ws.DeleteTemplate(ctx, tpl.ID); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	if _, err := ws.GetTemplate(ctx, tpl.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("deleted template lookup err = %v, want ErrNotFound", err)
	}
	doc, err := ws.Store.GetDocument(ctx, result.Document.ID)
	if err != nil {
		t.Fatalf("document must survive template deletion: %v", err)
	}
	if doc.TemplateID == nil || *doc.TemplateID != tpl.ID || doc.TemplateVersion != 1 {
		t.Fatalf("document snapshot = %v/%d, want dangling %d/1", doc.TemplateID, doc.TemplateVersion, tpl.ID)
	}
	facts, err := ws.Store.ListFactsByDocument(ctx, doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 3 {
		t.Fatalf("facts after template deletion = %d, want 3", len(facts))
	}
	// The platform is back to having no active membership template.
	if _, err := ws.ImportFile(ctx, source.ID, tpl.ID, path); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("import with deleted template err = %v, want ErrNotFound", err)
	}
}
