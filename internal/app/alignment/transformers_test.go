package alignment

import (
	"testing"
)

func runChain(t *testing.T, cfg MappingConfig, key, value string, names ...string) (string, error) {
	t.Helper()
	set, err := buildTransformers(cfg)
	if err != nil {
		t.Fatalf("build transformers: %v", err)
	}
	for _, name := range names {
		out, err := set[name](key, value)
		if err != nil {
			return "", err
		}
		value = out
	}
	return value, nil
}

func TestTransformTrim(t *testing.T) {
	got, err := runChain(t, MappingConfig{}, "k", "  uid-10001 \n", "trim")
	if err != nil || got != "uid-10001" {
		t.Fatalf("trim = %q, %v", got, err)
	}
}

func TestTransformStripQuotes(t *testing.T) {
	cases := []struct{ in, want string }{
		{`"260505170605000460"`, "260505170605000460"},
		{"'435167587794147", "435167587794147"},
		{`'quoted'`, "quoted"},
		{`plain`, "plain"},
		{`"unbalanced`, `"unbalanced`},
	}
	for _, c := range cases {
		got, err := runChain(t, MappingConfig{}, "k", c.in, "strip_quotes")
		if err != nil {
			t.Fatalf("strip_quotes(%q): %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("strip_quotes(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTransformParseDate(t *testing.T) {
	cases := []struct{ in, want string }{
		{"2026-05-01 10:00:00", "2026-05-01T10:00:00Z"},
		{"2026-05-01", "2026-05-01"},
		{"2026/05/01 10:00:00", "2026-05-01T10:00:00Z"},
		{"2026/5/1", "2026-05-01"},
		{"2026-5-2 9:05", "2026-05-02T09:05:00Z"},
		{"2026-05-01T10:00:00Z", "2026-05-01T10:00:00Z"},
		{"20260501", "2026-05-01"},
		{"45292", "2024-01-01"},             // plain serial day
		{"45292.5", "2024-01-01T12:00:00Z"}, // fractional serial keeps the time
		{"1", "1900-01-01"},                 // early-serial branch skips fictitious 1900-02-29
	}
	for _, c := range cases {
		got, err := runChain(t, MappingConfig{}, "k", c.in, "parseDate")
		if err != nil {
			t.Fatalf("parseDate(%q): %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("parseDate(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if _, err := runChain(t, MappingConfig{}, "k", "not a date", "parseDate"); err == nil {
		t.Fatal("expected error for unparseable date")
	}
}

func TestTransformMapEnum(t *testing.T) {
	cfg := MappingConfig{EnumMaps: map[string]map[string]string{
		"membership.level": {"总督": "governor", "提督": "admiral"},
	}}
	got, err := runChain(t, cfg, "membership.level", "总督", "mapEnum")
	if err != nil || got != "governor" {
		t.Fatalf("mapEnum = %q, %v", got, err)
	}
	if _, err := runChain(t, cfg, "membership.level", "舰长", "mapEnum"); err == nil {
		t.Fatal("expected error for unmapped value")
	}
	if _, err := runChain(t, cfg, "shipment.carrier_name", "申通快递", "mapEnum"); err == nil {
		t.Fatal("expected error when key has no enumMaps table")
	}
}

func TestTransformNormalizePhone(t *testing.T) {
	cases := []struct{ in, want string }{
		{"138 0000 0001", "13800000001"},
		{"138-0000-0002", "13800000002"},
		{"+8613800000003", "13800000003"},
		{"8613800000004", "13800000004"},
		{"+4915112345678", "+4915112345678"}, // other country prefix survives
		{" (021) 6543 ", "0216543"},
		// Fullwidth separators strip like their ASCII twins: ideographic
		// space, fullwidth hyphen, fullwidth parentheses.
		{"+86　138－0013－（8000）", "13800138000"},
	}
	for _, c := range cases {
		got, err := runChain(t, MappingConfig{}, "recipient.phone", c.in, "normalizePhone")
		if err != nil {
			t.Fatalf("normalizePhone(%q): %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("normalizePhone(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	// Known limitation: fullwidth digits pass through untouched instead of
	// folding back to ASCII digits.
	fullDigits := string([]rune{'\uff11', '\uff13', '\uff18', '\uff10', '\uff10', '\uff11', '\uff13', '\uff18', '\uff10', '\uff10', '\uff10', '\uff10'})
	gotFull, errFull := runChain(t, MappingConfig{}, "recipient.phone", fullDigits, "normalizePhone")
	if errFull != nil || gotFull != fullDigits {
		t.Fatalf("fullwidth digits must pass through untouched, got %q err %v", gotFull, errFull)
	}

	// Halfwidth and fullwidth spellings of the same number must collapse to
	// the exact same national number.
	pairs := [][2]string{
		{"+86 138-0013-(8000)", "+86　138－0013－（8000）"},
		{"86 - 139 - 0000 - (0001)", "86\u3000－\u3000139\u3000—\u30000000\u3000（0001）"},
	}
	for _, p := range pairs {
		half, err := runChain(t, MappingConfig{}, "recipient.phone", p[0], "normalizePhone")
		if err != nil {
			t.Fatalf("normalizePhone(%q): %v", p[0], err)
		}
		full, err := runChain(t, MappingConfig{}, "recipient.phone", p[1], "normalizePhone")
		if err != nil {
			t.Fatalf("normalizePhone(%q): %v", p[1], err)
		}
		if half == "" || half != full {
			t.Fatalf("halfwidth %q -> %q, fullwidth %q -> %q; must match", p[0], half, p[1], full)
		}
	}
}

func TestTransformJoinAddress(t *testing.T) {
	got, err := runChain(t, MappingConfig{}, "recipient.address_line1", " 江苏省 \x1f\x1f南京市\x1f 栖霞区 \x1f仙林大道1号", "joinAddress")
	if err != nil {
		t.Fatalf("joinAddress: %v", err)
	}
	if got != "江苏省南京市栖霞区仙林大道1号" {
		t.Fatalf("joinAddress = %q", got)
	}
	got, err = runChain(t, MappingConfig{}, "recipient.address_line1", "single-part", "joinAddress")
	if err != nil || got != "single-part" {
		t.Fatalf("joinAddress single part = %q, %v", got, err)
	}
}

func TestBuildTransformers_RejectsUnknownAndSplitInChain(t *testing.T) {
	if _, err := buildTransformers(MappingConfig{Transforms: map[string][]string{"k": {"explode"}}}); err == nil {
		t.Fatal("expected error for unknown transformer")
	}
	if _, err := buildTransformers(MappingConfig{Transforms: map[string][]string{"k": {"splitSkuQuantity"}}}); err == nil {
		t.Fatal("expected error for splitSkuQuantity in a chain")
	}
}

func TestParseSkuSegment(t *testing.T) {
	sku, title, qty, err := parseSkuSegment("206068021_设计师款透明单插立牌-底座可印刷15cm * 2")
	if err != nil {
		t.Fatalf("parse segment: %v", err)
	}
	if sku != "206068021" || title != "设计师款透明单插立牌-底座可印刷15cm" || qty != 2 {
		t.Fatalf("segment = %q / %q / %d", sku, title, qty)
	}
	sku, title, qty, err = parseSkuSegment("NOSKU_TITLE-ONLY")
	if err != nil || sku != "NOSKU" || title != "TITLE-ONLY" || qty != 1 {
		t.Fatalf("underscore-free segment = %q / %q / %d, %v", sku, title, qty, err)
	}
	if _, _, _, err := parseSkuSegment("sku_x * many"); err == nil {
		t.Fatal("expected error for non-numeric quantity")
	}
	if _, _, _, err := parseSkuSegment("sku_x * 0"); err == nil {
		t.Fatal("expected error for zero quantity")
	}
}
