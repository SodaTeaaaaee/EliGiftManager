package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// TestImportFile_RetailWithoutIdentityCreatesProfileFromRecipient covers the
// bilibili order export shape: no buyer uid, a buyer nickname, and recipient
// columns. The first order creates a profile named after the nickname with the
// recipient as its default address; a second order with the same recipient
// name + phone reuses that profile and adds the new address as non-default;
// a third with a different recipient creates another profile.
func TestImportFile_RetailWithoutIdentityCreatesProfileFromRecipient(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	seedBuiltins(t, ws)
	source := platformByKind(t, ws, domain.PlatformKindSource)
	tpl := cloneBuiltinTemplate(t, ws, source.ID, DocumentTypeOrderExport)

	path := writeTempFixture(t, "orders.csv",
		"\uFEFF订单号,商品名称,规格,数量,订单价格（含运费）,订单状态,付款时间,最晚发货时间,买家昵称,买家留言,收货人姓名,联系电话,收货地址\n"+
			"7005781651641387,【福利款】圆形透扇,--,2,19.80,待发货,2026-05-10 00:42:07,2026-06-09 00:42:07,Saushka,,钱先生,137 0000 0001,上海市 上海市 宝山区 一号\n"+
			"7005781651641388,【福利款】圆形透扇,--,1,9.90,待发货,2026-05-10 00:43:55,2026-06-09 00:43:55,,,钱先生,13700000001,江苏省 镇江市 二号\n"+
			"7003261006501158,【福利款】圆形透扇,--,2,19.80,待发货,2026-05-10 01:41:05,2026-06-09 01:41:05,,,郭天麒,17600000002,湖南省 长沙市 三号\n")
	result, err := ws.ImportFile(ctx, source.ID, tpl.ID, path)
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}
	if result.FactsCreated != 3 || result.LinesCreated != 3 || len(result.Issues) != 0 {
		t.Fatalf("result = %+v", result)
	}

	customers, err := ws.ListCustomers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(customers) != 2 {
		t.Fatalf("customers = %+v, want 2 (钱先生's orders share one profile)", customers)
	}
	byName := map[string]domain.CustomerProfile{}
	for _, c := range customers {
		byName[c.DisplayName] = c
	}
	saushka, ok := byName["Saushka"]
	if !ok {
		t.Fatalf("profile named after the buyer nickname missing: %+v", customers)
	}
	guo, ok := byName["郭天麒"]
	if !ok {
		t.Fatalf("profile named after the recipient (no nickname) missing: %+v", customers)
	}

	addrs, err := ws.ListAddresses(ctx, saushka.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 2 {
		t.Fatalf("Saushka addresses = %+v, want 2 distinct address lines", addrs)
	}
	if !addrs[0].IsDefault || addrs[1].IsDefault {
		t.Fatalf("first imported address must be the default: %+v", addrs)
	}
	if addrs[0].RecipientName != "钱先生" || addrs[0].Phone != "13700000001" || addrs[0].AddressLine1 != "上海市 上海市 宝山区 一号" {
		t.Fatalf("first address = %+v (phone must be normalized)", addrs[0])
	}
	if addrs[1].AddressLine1 != "江苏省 镇江市 二号" {
		t.Fatalf("second address = %+v", addrs[1])
	}
	guoAddrs, err := ws.ListAddresses(ctx, guo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(guoAddrs) != 1 || !guoAddrs[0].IsDefault || guoAddrs[0].Phone != "17600000002" {
		t.Fatalf("郭天麒 addresses = %+v", guoAddrs)
	}

	// The facts point at their profiles, the lines carry the title as alias
	// id, and the inbox does not flag anything as unattached.
	facts, err := ws.Store.ListFactsByDocument(ctx, result.Document.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range facts {
		if f.CustomerProfileID == nil || f.PlatformIdentityID != nil {
			t.Fatalf("fact %+v must have a profile and no identity", f)
		}
		if f.SourceCreatedAt == nil {
			t.Fatalf("fact %s must carry the payment time", f.StableExternalID)
		}
		lines, err := ws.Store.ListFactLines(ctx, f.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) != 1 || lines[0].ExternalSKU != "【福利款】圆形透扇" || lines[0].ExternalTitle != "【福利款】圆形透扇" {
			t.Fatalf("lines = %+v", lines)
		}
	}
	rows, err := ws.ListInboxRows(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Unattached {
			t.Fatalf("row %+v must not be unattached", r.Fact)
		}
	}

	// Assigning a line yields a result with the profile's default address, so
	// the retail order is not blocked on 收件信息不可用.
	wave, err := ws.CreateWave(ctx, "w", "")
	if err != nil {
		t.Fatal(err)
	}
	firstLines, err := ws.Store.ListFactLines(ctx, facts[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.AssignLines(ctx, wave.ID, []uint{firstLines[0].ID}); err != nil {
		t.Fatalf("AssignLines: %v", err)
	}
	views, err := ws.ListResultViews(ctx, wave.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 {
		t.Fatalf("views = %d, want 1", len(views))
	}
	for _, b := range views[0].Blocks {
		if b == domain.BlockUnusableAddress || b == domain.BlockIdentityUnattached {
			t.Fatalf("retail result blocked on %s despite imported recipient: %+v", b, views[0])
		}
	}
	if views[0].Result.Address.RecipientName != "钱先生" {
		t.Fatalf("result address = %+v", views[0].Result.Address)
	}

	// Re-importing the same file changes nothing on the profile side.
	if _, err := ws.ImportFile(ctx, source.ID, tpl.ID, path); err != nil {
		t.Fatal(err)
	}
	customers, err = ws.ListCustomers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(customers) != 2 {
		t.Fatalf("customers after re-import = %d, want 2", len(customers))
	}
	addrs, err = ws.ListAddresses(ctx, saushka.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 2 {
		t.Fatalf("addresses after re-import = %d, want 2", len(addrs))
	}
}

// TestIngest_IdentityWithRecipientLandsAddressOnProfile covers a source that
// carries both an identity and recipient data: the auto-created profile gets
// the address, and later facts for the same identity add addresses without
// duplicating identical ones. A later AttachIdentity to another profile moves
// the fact and its unfrozen results along.
func TestIngest_IdentityWithRecipientLandsAddressOnProfile(t *testing.T) {
	f := newRevisionFixture(t)
	ws, ctx := f.ws, f.ctx
	recipient := &domain.AddressSnapshot{RecipientName: "Alice Zhang", Phone: "13800000000", AddressLine1: "世纪大道100号"}
	f.ingest(t, IngestFactInput{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "ADDR-ORD-1",
		IdentityValue:    "uid-addr",
		DisplayName:      "Alice",
		Recipient:        recipient,
		Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 1}},
	})
	f.ingest(t, IngestFactInput{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "ADDR-ORD-2",
		IdentityValue:    "uid-addr",
		Recipient:        recipient,
		Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 2}},
	})
	ident, err := ws.Store.FindIdentity(ctx, f.source.ID, string(domain.IdentityTypePlatformUID), NormalizeIdentity("uid-addr"))
	if err != nil {
		t.Fatal(err)
	}
	if ident.CustomerProfileID == nil {
		t.Fatal("identity must be attached to the auto-created profile")
	}
	profile, err := ws.GetCustomer(ctx, *ident.CustomerProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if profile.DisplayName != "Alice" {
		t.Fatalf("profile name = %q, want the imported display name", profile.DisplayName)
	}
	addrs, err := ws.ListAddresses(ctx, profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || !addrs[0].IsDefault || addrs[0].RecipientName != "Alice Zhang" {
		t.Fatalf("addresses = %+v, want one default address (identical recipients dedupe)", addrs)
	}
	// Recipient data on a fact with a resolved identity never creates a second
	// profile, even when another profile already owns the same recipient.
	customers, err := ws.ListCustomers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(customers) != 2 { // fixture customer + auto profile
		t.Fatalf("customers = %+v, want fixture customer + 1 auto profile", customers)
	}

	// Assign both lines, then re-point the identity to the fixture customer
	// (who has no address): results follow the new profile and lose the
	// imported address, which now blocks them.
	facts := mustFactsByPlatform(t, ws, f.source.ID)
	var lineIDs []uint
	for _, fact := range facts {
		lines, err := ws.Store.ListFactLines(ctx, fact.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, ln := range lines {
			lineIDs = append(lineIDs, ln.ID)
		}
	}
	if err := ws.AssignLines(ctx, f.wave.ID, lineIDs); err != nil {
		t.Fatal(err)
	}
	views, err := ws.ListResultViews(ctx, f.wave.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range views {
		if v.WorkState != domain.WorkStateReady || v.Result.Address.RecipientName != "Alice Zhang" {
			t.Fatalf("pre-attach view = %+v, want ready with the imported address", v)
		}
	}
	if err := ws.AttachIdentity(ctx, ident.ID, f.cust.ID); err != nil {
		t.Fatalf("AttachIdentity: %v", err)
	}
	for _, fact := range mustFactsByPlatform(t, ws, f.source.ID) {
		if fact.CustomerProfileID == nil || *fact.CustomerProfileID != f.cust.ID {
			t.Fatalf("fact %s customer = %v, want re-pointed to %d", fact.StableExternalID, fact.CustomerProfileID, f.cust.ID)
		}
	}
	views, err = ws.ListResultViews(ctx, f.wave.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range views {
		if v.Result.CustomerProfileID == nil || *v.Result.CustomerProfileID != f.cust.ID {
			t.Fatalf("result customer = %v, want %d", v.Result.CustomerProfileID, f.cust.ID)
		}
		if v.WorkState != domain.WorkStateBlocked {
			t.Fatalf("post-attach view = %+v, want blocked on the new profile's missing address", v)
		}
	}
}
