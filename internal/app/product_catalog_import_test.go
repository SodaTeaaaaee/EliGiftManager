package app

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/pathconfine"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
)

func TestParseMappingRules_ImageLayoutDefaults(t *testing.T) {
	t.Parallel()

	raw := `{
		"version": 2,
		"mode": "header",
		"hasHeader": true,
		"columns": {"product.factory_sku": "SKU", "product.name": "Name"},
		"imageLayout": {
			"enabled": true,
			"coverDir": "covers",
			"detailDir": "details"
		}
	}`
	rules, err := ParseMappingRules(raw)
	if err != nil {
		t.Fatalf("ParseMappingRules: %v", err)
	}
	if rules.ImageLayout == nil {
		t.Fatal("expected ImageLayout")
	}
	l := rules.ImageLayout
	if !l.Enabled {
		t.Error("enabled")
	}
	if l.MatchField != "product.name" {
		t.Errorf("MatchField = %q", l.MatchField)
	}
	if l.NamePattern != "{match}#{nn}" {
		t.Errorf("NamePattern = %q", l.NamePattern)
	}
	if l.CoverPick != "lowest_nn" {
		t.Errorf("CoverPick = %q", l.CoverPick)
	}
	if l.TabularGlob != "*.csv" {
		t.Errorf("TabularGlob = %q", l.TabularGlob)
	}
	if l.CoverDir != "covers" || l.DetailDir != "details" {
		t.Errorf("dirs: %+v", l)
	}
	if len(l.ImageExts) == 0 {
		t.Error("expected default ImageExts")
	}
}

func TestParseMappingRules_ImageLayoutOmitted(t *testing.T) {
	t.Parallel()

	raw := `{
		"version": 2,
		"mode": "header",
		"hasHeader": true,
		"columns": {"product.factory_sku": "SKU"}
	}`
	rules, err := ParseMappingRules(raw)
	if err != nil {
		t.Fatalf("ParseMappingRules: %v", err)
	}
	if rules.ImageLayout != nil {
		t.Fatalf("expected nil ImageLayout, got %+v", rules.ImageLayout)
	}
}

func TestImportProductCatalog_ZipWithImages(t *testing.T) {
	t.Parallel()

	// Mini catalog zip: CSV + cover/detail images matched by product name pattern.
	// Cover content is identical for Widget and Gadget → AssetStore hash dedup.
	coverBytes := []byte("cover-image-bytes-v1")
	detail2Bytes := []byte("detail-image-bytes-02")
	detail3Bytes := []byte("detail-image-bytes-03")
	csvBody := "SKU,Name,Kind\nWID-1,Widget,badge\nGAD-1,Gadget,badge\n"

	zipPath := filepath.Join(t.TempDir(), "catalog.zip")
	if err := writeTestZip(zipPath, map[string]string{
		"catalog.csv":      csvBody,
		"主图/Widget#01.png": string(coverBytes),
		"详情/Widget#02.png": string(detail2Bytes),
		"详情/Widget#03.png": string(detail3Bytes),
		"主图/Gadget#01.png": string(coverBytes),
	}); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	assetsRoot := t.TempDir()
	store := service.NewAssetStoreAt(assetsRoot)
	extractRoot := t.TempDir()

	masterRepo := newCatalogTestMasterRepo()
	profileRepo := newCatalogTestProfileRepo()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "factory",
		SourceSurface:                string(domain.SourceSurfaceFactory),
		SupportsImportProductCatalog: true,
		FactorySupplierPlatform:      "test-platform",
		ConnectorKey:                 "factory-a",
	})
	templateRepo := newCatalogTestTemplateRepo()
	bindingRepo := newCatalogTestBindingRepo()
	// Paths come from rules JSON — not hard-coded platform branches.
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "catalog-zip", DocumentType: "import_product_catalog", Format: "zip",
		MappingRules: `{
			"version": 2, "mode": "header", "hasHeader": true,
			"columns": {
				"product.factory_sku": "SKU",
				"product.name": "Name",
				"product.product_kind": "Kind"
			},
			"imageLayout": {
				"enabled": true,
				"matchField": "product.name",
				"coverDir": "主图",
				"detailDir": "详情",
				"namePattern": "{match}#{nn}",
				"coverPick": "lowest_nn",
				"tabularGlob": "*.csv",
				"imageExts": [".png", ".jpg"]
			}
		}`,
	}
	if err := templateRepo.Create(context.Background(), tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	if err := bindingRepo.Create(context.Background(), &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: 1, DocumentType: "import_product_catalog", TemplateID: tmpl.ID, IsDefault: true,
	}); err != nil {
		t.Fatalf("create binding: %v", err)
	}
	mapping := NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)

	uc := NewProductUseCase(masterRepo, nil, nil)
	uc = WithCatalogImportDeps(uc, mapping, profileRepo, store)
	uc.(*productUseCase).catalog.extractRoot = extractRoot

	result, err := uc.ImportProductCatalog(context.Background(), dto.ImportProductCatalogInput{
		IntegrationProfileID: 1,
		ImportMode:           "reject_all",
		FilePath:             zipPath,
	})
	if err != nil {
		t.Fatalf("ImportProductCatalog: %v", err)
	}
	if result.CreatedCount != 2 || result.ErrorCount != 0 {
		t.Fatalf("result: %+v errors=%v", result, result.Errors)
	}

	var widget, gadget *dto.ProductMasterDTO
	for i := range result.Masters {
		m := &result.Masters[i]
		switch m.FactorySKU {
		case "WID-1":
			widget = m
		case "GAD-1":
			gadget = m
		}
	}
	if widget == nil || gadget == nil {
		t.Fatalf("masters missing: %+v", result.Masters)
	}

	// Cover path written + content-addressed layout.
	sum := sha256.Sum256(coverBytes)
	hash := hex.EncodeToString(sum[:])
	wantCover := path.Join("products", hash[:2], hash+".png")
	if widget.CoverImagePath != wantCover {
		t.Errorf("widget cover = %q, want %q", widget.CoverImagePath, wantCover)
	}
	if gadget.CoverImagePath != wantCover {
		t.Errorf("gadget cover should dedup to same hash path, got %q want %q", gadget.CoverImagePath, wantCover)
	}

	// Detail paths ordered by nn ascending.
	var details []string
	if err := json.Unmarshal([]byte(widget.DetailImagePaths), &details); err != nil {
		t.Fatalf("detail json: %v (%q)", err, widget.DetailImagePaths)
	}
	if len(details) != 2 {
		t.Fatalf("widget details = %v, want 2", details)
	}
	sum2 := sha256.Sum256(detail2Bytes)
	hash2 := hex.EncodeToString(sum2[:])
	wantD2 := path.Join("products", hash2[:2], hash2+".png")
	sum3 := sha256.Sum256(detail3Bytes)
	hash3 := hex.EncodeToString(sum3[:])
	wantD3 := path.Join("products", hash3[:2], hash3+".png")
	if details[0] != wantD2 || details[1] != wantD3 {
		t.Errorf("details order/paths = %v, want [%s %s]", details, wantD2, wantD3)
	}

	// Hash dedup: coverBytes stored once on disk.
	var files []string
	_ = filepath.WalkDir(filepath.Join(assetsRoot, "products"), func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".tmp") {
			return nil
		}
		files = append(files, p)
		return nil
	})
	// cover (shared) + detail2 + detail3 = 3 unique files
	if len(files) != 3 {
		t.Fatalf("expected 3 unique asset files (hash dedup), got %d: %v", len(files), files)
	}

	// Extract dir cleaned up.
	entries, _ := os.ReadDir(extractRoot)
	if len(entries) != 0 {
		t.Errorf("extract root should be empty after cleanup, got %v", entries)
	}
}

func TestImportProductCatalog_ZipWithoutImageLayout(t *testing.T) {
	t.Parallel()

	csvBody := "SKU,Name\nZ-1,ZipOnly\n"
	zipPath := filepath.Join(t.TempDir(), "plain.zip")
	if err := writeTestZip(zipPath, map[string]string{"data.csv": csvBody}); err != nil {
		t.Fatal(err)
	}

	masterRepo := newCatalogTestMasterRepo()
	profileRepo := newCatalogTestProfileRepo()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "factory", ConnectorKey: "plat",
		SourceSurface: string(domain.SourceSurfaceFactory), SupportsImportProductCatalog: true,
	})
	templateRepo := newCatalogTestTemplateRepo()
	bindingRepo := newCatalogTestBindingRepo()
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "catalog", DocumentType: "import_product_catalog", Format: "csv",
		MappingRules: `{
			"version": 2, "mode": "header", "hasHeader": true,
			"columns": {"product.factory_sku": "SKU", "product.name": "Name"}
		}`,
	}
	_ = templateRepo.Create(context.Background(), tmpl)
	_ = bindingRepo.Create(context.Background(), &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: 1, DocumentType: "import_product_catalog", TemplateID: tmpl.ID, IsDefault: true,
	})
	mapping := NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)

	uc := NewProductUseCase(masterRepo, nil, nil)
	uc = WithCatalogImportDeps(uc, mapping, profileRepo, service.NewAssetStoreAt(t.TempDir()))
	uc.(*productUseCase).catalog.extractRoot = t.TempDir()

	result, err := uc.ImportProductCatalog(context.Background(), dto.ImportProductCatalogInput{
		IntegrationProfileID: 1,
		ImportMode:           "reject_all",
		FilePath:             zipPath,
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.CreatedCount != 1 {
		t.Fatalf("created: %+v", result)
	}
	if result.Masters[0].CoverImagePath != "" {
		t.Errorf("unexpected cover: %q", result.Masters[0].CoverImagePath)
	}
}

// TestImportProductCatalog_ZipWithNestedRootDir covers platform exports that wrap
// CSV + image dirs under a single top-level folder, while CoverDir stays "主图".
// Layout: RootName/主图/, RootName/详情/, RootName/a.csv
func TestImportProductCatalog_ZipWithNestedRootDir(t *testing.T) {
	t.Parallel()

	coverBytes := []byte("nested-cover-bytes-v1")
	detailBytes := []byte("nested-detail-bytes-02")
	csvBody := "SKU,Name,Kind\nNEST-1,NestedWidget,badge\n"
	rootName := "从工厂平台导出-商品列表"

	zipPath := filepath.Join(t.TempDir(), "nested-catalog.zip")
	if err := writeTestZip(zipPath, map[string]string{
		path.Join(rootName, "a.csv"):                     csvBody,
		path.Join(rootName, "主图", "NestedWidget#01.png"): string(coverBytes),
		path.Join(rootName, "详情", "NestedWidget#02.png"): string(detailBytes),
	}); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	assetsRoot := t.TempDir()
	store := service.NewAssetStoreAt(assetsRoot)
	extractRoot := t.TempDir()

	masterRepo := newCatalogTestMasterRepo()
	profileRepo := newCatalogTestProfileRepo()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "factory",
		SourceSurface:                string(domain.SourceSurfaceFactory),
		SupportsImportProductCatalog: true,
		FactorySupplierPlatform:      "test-platform",
		ConnectorKey:                 "factory-a",
	})
	templateRepo := newCatalogTestTemplateRepo()
	bindingRepo := newCatalogTestBindingRepo()
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "catalog-zip-nested", DocumentType: "import_product_catalog", Format: "zip",
		MappingRules: `{
			"version": 2, "mode": "header", "hasHeader": true,
			"columns": {
				"product.factory_sku": "SKU",
				"product.name": "Name",
				"product.product_kind": "Kind"
			},
			"imageLayout": {
				"enabled": true,
				"matchField": "product.name",
				"coverDir": "主图",
				"detailDir": "详情",
				"namePattern": "{match}#{nn}",
				"coverPick": "lowest_nn",
				"tabularGlob": "*.csv",
				"imageExts": [".png", ".jpg"]
			}
		}`,
	}
	if err := templateRepo.Create(context.Background(), tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	if err := bindingRepo.Create(context.Background(), &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: 1, DocumentType: "import_product_catalog", TemplateID: tmpl.ID, IsDefault: true,
	}); err != nil {
		t.Fatalf("create binding: %v", err)
	}
	mapping := NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)

	uc := NewProductUseCase(masterRepo, nil, nil)
	uc = WithCatalogImportDeps(uc, mapping, profileRepo, store)
	uc.(*productUseCase).catalog.extractRoot = extractRoot

	result, err := uc.ImportProductCatalog(context.Background(), dto.ImportProductCatalogInput{
		IntegrationProfileID: 1,
		ImportMode:           "reject_all",
		FilePath:             zipPath,
	})
	if err != nil {
		t.Fatalf("ImportProductCatalog: %v", err)
	}
	if result.CreatedCount != 1 || result.ErrorCount != 0 {
		t.Fatalf("result: %+v errors=%v", result, result.Errors)
	}
	if len(result.Masters) != 1 {
		t.Fatalf("masters: %+v", result.Masters)
	}
	m := result.Masters[0]
	if m.FactorySKU != "NEST-1" {
		t.Errorf("sku = %q", m.FactorySKU)
	}

	sum := sha256.Sum256(coverBytes)
	hash := hex.EncodeToString(sum[:])
	wantCover := path.Join("products", hash[:2], hash+".png")
	if m.CoverImagePath != wantCover {
		t.Errorf("cover = %q, want %q", m.CoverImagePath, wantCover)
	}

	var details []string
	if err := json.Unmarshal([]byte(m.DetailImagePaths), &details); err != nil {
		t.Fatalf("detail json: %v (%q)", err, m.DetailImagePaths)
	}
	if len(details) != 1 {
		t.Fatalf("details = %v, want 1", details)
	}
	sumD := sha256.Sum256(detailBytes)
	hashD := hex.EncodeToString(sumD[:])
	wantDetail := path.Join("products", hashD[:2], hashD+".png")
	if details[0] != wantDetail {
		t.Errorf("detail = %q, want %q", details[0], wantDetail)
	}
}

func TestResolveCatalogContentRoot_NestedUniqueParent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	parent := filepath.Join(root, "从工厂平台导出-商品列表")
	if err := os.MkdirAll(filepath.Join(parent, "主图"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parent, "详情"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "a.csv"), []byte("SKU\n1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	layout := &CatalogImageLayout{CoverDir: "主图", DetailDir: "详情"}
	got := resolveCatalogContentRoot(root, layout)
	if got != parent {
		t.Errorf("content root = %q, want %q", got, parent)
	}

	// Flat layout still resolves to extract root.
	flat := t.TempDir()
	if err := os.MkdirAll(filepath.Join(flat, "主图"), 0o755); err != nil {
		t.Fatal(err)
	}
	gotFlat := resolveCatalogContentRoot(flat, layout)
	if gotFlat != flat {
		t.Errorf("flat content root = %q, want %q", gotFlat, flat)
	}
}

func TestFindTabularInDir_NestedCSV(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "RootName")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	csvPath := filepath.Join(nested, "a.csv")
	if err := os.WriteFile(csvPath, []byte("SKU\n1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := findTabularInDir(root, "*.csv")
	if err != nil {
		t.Fatalf("findTabularInDir: %v", err)
	}
	if found != csvPath {
		t.Errorf("found = %q, want %q", found, csvPath)
	}
}

func TestNamePatternRegex(t *testing.T) {
	t.Parallel()

	re, err := namePatternRegex("{match}#{nn}", "Widget")
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("Widget#01") {
		t.Error("should match Widget#01")
	}
	if re.MatchString("Widget") {
		t.Error("should not match without nn")
	}
	if re.MatchString("Other#01") {
		t.Error("should not match other product")
	}
	m := re.FindStringSubmatch("Widget#12")
	if len(m) < 2 || m[1] != "12" {
		t.Errorf("capture = %v", m)
	}
}

func writeTestZip(zipPath string, files map[string]string) error {
	ordered := make([][2]string, 0, len(files))
	for name, body := range files {
		ordered = append(ordered, [2]string{name, body})
	}
	return writeTestZipOrdered(zipPath, ordered)
}

func writeTestZipOrdered(zipPath string, files [][2]string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	for _, item := range files {
		fw, err := w.Create(item[0])
		if err != nil {
			_ = w.Close()
			return err
		}
		if _, err := fw.Write([]byte(item[1])); err != nil {
			_ = w.Close()
			return err
		}
	}
	return w.Close()
}

func TestRejectUnsafeRelPath_WindowsAbsUNCDotDot(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
	}{
		{"drive_backslash", `C:\Windows\Temp`},
		{"drive_slash", `C:/Windows/Temp`},
		{"drive_rel", `C:foo`},
		{"unc_backslash", `\\server\share`},
		{"unc_slash", `//server/share/covers`},
		{"dotdot", `..\..`},
		{"dotdot_nested", `foo\..\..\secret`},
		{"dotdot_slash", `../../etc`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := pathconfine.JoinUnder(t.TempDir(), tc.path); err == nil {
				t.Fatalf("expected reject %q", tc.path)
			}
		})
	}
	if pathconfine.LooksAbsolute("covers") || pathconfine.HasDotDot("covers") {
		t.Fatal("safe relative path rejected")
	}
}

func TestConfinedJoin_StaysInsideRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	got, err := pathconfine.JoinUnder(root, "covers")
	if err != nil {
		t.Fatalf("JoinUnder: %v", err)
	}
	if _, err := pathconfine.Confine(root, got); err != nil {
		t.Fatal(err)
	}
	if _, err := pathconfine.JoinUnder(root, `C:\Windows\Temp`); err == nil {
		t.Fatal("expected reject absolute CoverDir")
	}
	if _, err := pathconfine.JoinUnder(root, `\\server\share`); err == nil {
		t.Fatal("expected reject UNC")
	}
	if _, err := pathconfine.JoinUnder(root, `..\..`); err == nil {
		t.Fatal("expected reject ..")
	}
}

func TestAttachCatalogImages_RejectsEscapingCoverAndDetailDir(t *testing.T) {
	t.Parallel()

	imageRoot := t.TempDir()
	outsideDir := filepath.Dir(imageRoot)
	secret := filepath.Join(outsideDir, "Widget#01.png")
	if err := os.WriteFile(secret, []byte("SECRET-OUTSIDE"), 0o644); err != nil {
		t.Fatal(err)
	}
	storeRoot := t.TempDir()
	store := service.NewAssetStoreAt(storeRoot)
	master := &domain.ProductMaster{Name: "Widget", FactorySKU: "W-1"}

	cases := []struct {
		name   string
		cover  string
		detail string
	}{
		{"cover_windows_abs", `C:\Windows\Temp`, ""},
		{"detail_windows_abs", "", `C:\Windows\Temp`},
		{"cover_unc", `\\server\share\covers`, ""},
		{"detail_unc", "", `\\server\share\details`},
		{"cover_dotdot", `..\..`, ""},
		{"detail_dotdot", "", `..\..`},
		{"cover_parent", `..`, ""},
		{"detail_parent", "", `..`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := *master
			layout := &CatalogImageLayout{
				Enabled:     true,
				CoverDir:    tc.cover,
				DetailDir:   tc.detail,
				NamePattern: "{match}#{nn}",
				CoverPick:   "lowest_nn",
				ImageExts:   []string{".png"},
			}
			err := attachCatalogImages(&m, layout, imageRoot, store)
			if err == nil {
				t.Fatal("expected confinement error")
			}
			if m.CoverImagePath != "" || m.DetailImagePaths != "" {
				t.Fatalf("must not ingest escaped images: cover=%q details=%q", m.CoverImagePath, m.DetailImagePaths)
			}
		})
	}
}

func TestFindTabularInDir_RejectsPathEscape(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "escape.csv")
	if err := os.WriteFile(outside, []byte("SKU\n1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ok.csv"), []byte("SKU\n2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []string{
		`..\escape.csv`,
		`C:\Windows\*.csv`,
		`\\server\share\*.csv`,
		`//server/share/*.csv`,
		`..\..\*.csv`,
	}
	for _, glob := range cases {
		t.Run(glob, func(t *testing.T) {
			t.Parallel()
			got, err := findTabularInDir(root, glob)
			if err == nil {
				t.Fatalf("expected reject glob %q, got %q", glob, got)
			}
			if got != "" {
				t.Fatalf("escaped match must not be returned: %q", got)
			}
		})
	}

	found, err := findTabularInDir(root, "*.csv")
	if err != nil {
		t.Fatalf("safe glob: %v", err)
	}
	if found != filepath.Join(root, "ok.csv") {
		t.Fatalf("found = %q", found)
	}
}

func TestExtractCatalogZip_TooManyEntries(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "many.zip")
	files := make([][2]string, 0, 8)
	for i := 0; i < 8; i++ {
		files = append(files, [2]string{fmt.Sprintf("f%d.txt", i), "x"})
	}
	if err := writeTestZipOrdered(zipPath, files); err != nil {
		t.Fatal(err)
	}
	deps := &catalogImportDeps{
		extractRoot: t.TempDir(),
		zipLimits:   &catalogZipLimits{MaxEntries: 3, MaxFileBytes: 1024, MaxTotalBytes: 4096, MaxHashBytes: 1024},
	}
	dir, err := extractCatalogZip(zipPath, deps)
	if err == nil {
		t.Fatalf("expected too-many-entries error, extracted %q", dir)
	}
	if dir != "" {
		t.Fatalf("failed extract must not return dir, got %q", dir)
	}
}

func TestExtractCatalogZip_OversizedFile(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "big.zip")
	body := string(bytes.Repeat([]byte("a"), 200))
	if err := writeTestZip(zipPath, map[string]string{"big.bin": body}); err != nil {
		t.Fatal(err)
	}
	deps := &catalogImportDeps{
		extractRoot: t.TempDir(),
		zipLimits:   &catalogZipLimits{MaxEntries: 10, MaxFileBytes: 50, MaxTotalBytes: 10_000, MaxHashBytes: 50},
	}
	dir, err := extractCatalogZip(zipPath, deps)
	if err == nil {
		t.Fatalf("expected oversized-file error, extracted %q", dir)
	}
	if dir != "" {
		t.Fatalf("failed extract must not return dir, got %q", dir)
	}
}

func TestExtractCatalogZip_TotalSizeLimit(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "total.zip")
	if err := writeTestZipOrdered(zipPath, [][2]string{
		{"a.bin", string(bytes.Repeat([]byte("a"), 40))},
		{"b.bin", string(bytes.Repeat([]byte("b"), 40))},
	}); err != nil {
		t.Fatal(err)
	}
	deps := &catalogImportDeps{
		extractRoot: t.TempDir(),
		zipLimits:   &catalogZipLimits{MaxEntries: 10, MaxFileBytes: 100, MaxTotalBytes: 50, MaxHashBytes: 100},
	}
	dir, err := extractCatalogZip(zipPath, deps)
	if err == nil {
		t.Fatalf("expected total-size error, extracted %q", dir)
	}
	if dir != "" {
		t.Fatalf("failed extract must not return dir, got %q", dir)
	}
}

func TestExtractCatalogZip_ZipSlipRejected(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "slip.zip")
	if err := writeTestZipOrdered(zipPath, [][2]string{
		{"ok.csv", "SKU,Name\n1,A\n"},
		{"../evil.txt", "pwned"},
	}); err != nil {
		t.Fatal(err)
	}
	extractRoot := t.TempDir()
	deps := &catalogImportDeps{extractRoot: extractRoot}
	dir, err := extractCatalogZip(zipPath, deps)
	if err == nil {
		t.Fatalf("zip-slip must fail extract, got dir %q", dir)
	}
	if dir != "" {
		t.Fatalf("zip-slip must not return extract dir, got %q", dir)
	}
	evil := filepath.Join(extractRoot, "evil.txt")
	if _, statErr := os.Stat(evil); statErr == nil {
		t.Fatal("zip-slip wrote evil.txt outside extract dir")
	}
}

func TestExtractCatalogZip_SkipsWindowsIllegalImageNames(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-illegal filenames")
	}
	zipPath := filepath.Join(t.TempDir(), "star.zip")
	if err := writeTestZipOrdered(zipPath, [][2]string{
		{"ok.csv", "SKU,Name\n1,A\n"},
		{"covers/A6(107*150mm)#01.png", "img"},
	}); err != nil {
		t.Fatal(err)
	}
	deps := &catalogImportDeps{extractRoot: t.TempDir()}
	dir, err := extractCatalogZip(zipPath, deps)
	if err != nil {
		t.Fatalf("illegal image name must be skipped, not fail extract: %v", err)
	}
	if dir == "" {
		t.Fatal("expected extract dir")
	}
	if len(deps.extractWarnings) == 0 {
		t.Fatal("expected skip warning for * in filename")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "ok.csv")); statErr != nil {
		t.Fatalf("csv should be extracted: %v", statErr)
	}
}

func TestExtractCatalogZip_MemberFailureSurfacesError(t *testing.T) {
	// File named "collision" then "collision/nested.txt": mkdir of the parent
	// fails after the file is created — must not return extractDir, nil.
	zipPath := filepath.Join(t.TempDir(), "conflict.zip")
	if err := writeTestZipOrdered(zipPath, [][2]string{
		{"collision", "i-am-a-file"},
		{"collision/nested.csv", "SKU,Name\n1,A\n"},
	}); err != nil {
		t.Fatal(err)
	}
	deps := &catalogImportDeps{extractRoot: t.TempDir()}
	dir, err := extractCatalogZip(zipPath, deps)
	if err == nil {
		t.Fatalf("member mkdir/copy failure must surface, got dir %q", dir)
	}
	if dir != "" {
		t.Fatalf("failed extract must not return dir, got %q", dir)
	}
}

func TestImportProductCatalog_ZipSlipSurfacesError(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "slip-import.zip")
	if err := writeTestZipOrdered(zipPath, [][2]string{
		{"catalog.csv", "SKU,Name\nS-1,Slip\n"},
		{"../evil.txt", "pwned"},
	}); err != nil {
		t.Fatal(err)
	}

	masterRepo := newCatalogTestMasterRepo()
	profileRepo := newCatalogTestProfileRepo()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "factory", ConnectorKey: "plat",
		SourceSurface: string(domain.SourceSurfaceFactory), SupportsImportProductCatalog: true,
	})
	templateRepo := newCatalogTestTemplateRepo()
	bindingRepo := newCatalogTestBindingRepo()
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "catalog", DocumentType: "import_product_catalog", Format: "csv",
		MappingRules: `{
			"version": 2, "mode": "header", "hasHeader": true,
			"columns": {"product.factory_sku": "SKU", "product.name": "Name"}
		}`,
	}
	_ = templateRepo.Create(context.Background(), tmpl)
	_ = bindingRepo.Create(context.Background(), &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: 1, DocumentType: "import_product_catalog", TemplateID: tmpl.ID, IsDefault: true,
	})
	mapping := NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)
	uc := NewProductUseCase(masterRepo, nil, nil)
	uc = WithCatalogImportDeps(uc, mapping, profileRepo, service.NewAssetStoreAt(t.TempDir()))
	uc.(*productUseCase).catalog.extractRoot = t.TempDir()

	result, err := uc.ImportProductCatalog(context.Background(), dto.ImportProductCatalogInput{
		IntegrationProfileID: 1,
		ImportMode:           "reject_all",
		FilePath:             zipPath,
	})
	if err == nil {
		t.Fatalf("import must surface zip-slip extract error, result=%+v", result)
	}
}

func TestZipAssetMetadata_CapsHashing(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "hash.zip")
	body := string(bytes.Repeat([]byte("h"), 500))
	if err := writeTestZip(zipPath, map[string]string{"photo.png": body}); err != nil {
		t.Fatal(err)
	}
	_, err := zipCatalogAssetMetadata(zipPath, catalogZipLimits{
		MaxEntries: 10, MaxFileBytes: 1024, MaxTotalBytes: 4096, MaxHashBytes: 64,
	})
	if err == nil {
		t.Fatal("expected hash size cap error")
	}

	got, err := zipCatalogAssetMetadata(zipPath, catalogZipLimits{
		MaxEntries: 10, MaxFileBytes: 1024, MaxTotalBytes: 4096, MaxHashBytes: 1024,
	})
	if err != nil {
		t.Fatalf("hash within cap: %v", err)
	}
	if len(got) != 1 || got[0]["sha256"] == "" {
		t.Fatalf("metadata = %+v", got)
	}
	sum := sha256.Sum256([]byte(body))
	if got[0]["sha256"] != hex.EncodeToString(sum[:]) {
		t.Errorf("sha256 = %q", got[0]["sha256"])
	}
}

func TestSanitizeCatalogImageExts_Allowlist(t *testing.T) {
	t.Parallel()

	got := sanitizeCatalogImageExts([]string{".svg", ".html", ".json", ".png", "JPG", "exe"})
	if len(got) != 2 || got[0] != ".png" || got[1] != ".jpg" {
		t.Fatalf("sanitized = %v, want [.png .jpg]", got)
	}
	if len(sanitizeCatalogImageExts([]string{".svg", ".html"})) != 0 {
		t.Fatal("unsafe-only list must be empty")
	}
	def := sanitizeCatalogImageExts(nil)
	for _, ext := range def {
		if !isAllowedCatalogImageExt(ext) {
			t.Fatalf("default %q not allowlisted", ext)
		}
	}
}

func TestAttachCatalogImages_ImageExtAllowlist(t *testing.T) {
	t.Parallel()

	imageRoot := t.TempDir()
	coverDir := filepath.Join(imageRoot, "covers")
	if err := os.MkdirAll(coverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coverDir, "Widget#01.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coverDir, "Widget#02.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coverDir, "Widget#03.json"), []byte(`{"x":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	pngBytes := []byte("png-cover-bytes")
	if err := os.WriteFile(filepath.Join(coverDir, "Widget#04.png"), pngBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	store := service.NewAssetStoreAt(t.TempDir())
	master := &domain.ProductMaster{Name: "Widget", FactorySKU: "W-1"}
	layout := &CatalogImageLayout{
		Enabled:     true,
		CoverDir:    "covers",
		NamePattern: "{match}#{nn}",
		CoverPick:   "lowest_nn",
		ImageExts:   []string{".svg", ".html", ".json", ".png"},
	}
	if err := attachCatalogImages(master, layout, imageRoot, store); err != nil {
		t.Fatalf("attach: %v", err)
	}
	sum := sha256.Sum256(pngBytes)
	hash := hex.EncodeToString(sum[:])
	want := path.Join("products", hash[:2], hash+".png")
	if master.CoverImagePath != want {
		t.Fatalf("cover = %q, want png %q (svg/html/json must not be ingested)", master.CoverImagePath, want)
	}

	unsafeOnly := &domain.ProductMaster{Name: "Widget", FactorySKU: "W-2"}
	if err := attachCatalogImages(unsafeOnly, &CatalogImageLayout{
		Enabled: true, CoverDir: "covers", NamePattern: "{match}#{nn}",
		CoverPick: "lowest_nn", ImageExts: []string{".svg", ".html", ".json"},
	}, imageRoot, store); err != nil {
		t.Fatalf("unsafe-only attach: %v", err)
	}
	if unsafeOnly.CoverImagePath != "" {
		t.Fatalf("must not ingest svg/html/json cover, got %q", unsafeOnly.CoverImagePath)
	}
}

func TestImportProductCatalog_SampleCatalogZipIfPresent(t *testing.T) {
	sample := filepath.Join("..", "..", "SampleData", "工厂平台——柔造", "从工厂平台导出-商品列表.zip")
	if _, err := os.Stat(sample); err != nil {
		t.Skip("SampleData catalog zip not present")
	}

	masterRepo := newCatalogTestMasterRepo()
	profileRepo := newCatalogTestProfileRepo()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "factory", ConnectorKey: "plat",
		SourceSurface: string(domain.SourceSurfaceFactory), SupportsImportProductCatalog: true,
	})
	templateRepo := newCatalogTestTemplateRepo()
	bindingRepo := newCatalogTestBindingRepo()
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "catalog", DocumentType: "import_product_catalog", Format: "zip",
		MappingRules: CatalogDemoMappingRules,
	}
	_ = templateRepo.Create(context.Background(), tmpl)
	_ = bindingRepo.Create(context.Background(), &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: 1, DocumentType: "import_product_catalog", TemplateID: tmpl.ID, IsDefault: true,
	})
	mapping := NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)
	uc := NewProductUseCase(masterRepo, nil, nil)
	uc = WithCatalogImportDeps(uc, mapping, profileRepo, service.NewAssetStoreAt(t.TempDir()))
	uc.(*productUseCase).catalog.extractRoot = t.TempDir()

	result, err := uc.ImportProductCatalog(context.Background(), dto.ImportProductCatalogInput{
		IntegrationProfileID: 1,
		ImportMode:           "skip_invalid",
		FilePath:             sample,
	})
	if err != nil {
		t.Fatalf("sample catalog import error: %v", err)
	}
	if result.SuccessCount == 0 {
		t.Fatalf("sample catalog SuccessCount=0 errors=%+v total=%d", result.Errors, result.TotalProcessed)
	}
}

func TestImportProductCatalog_SkipInvalidAttachFailureDoesNotCountSuccess(t *testing.T) {
	t.Parallel()

	csvBody := "SKU,Name\nW-1,Widget\n"
	zipPath := filepath.Join(t.TempDir(), "catalog.zip")
	if err := writeTestZip(zipPath, map[string]string{"catalog.csv": csvBody}); err != nil {
		t.Fatal(err)
	}

	masterRepo := newCatalogTestMasterRepo()
	profileRepo := newCatalogTestProfileRepo()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "factory", ConnectorKey: "plat",
		SourceSurface: string(domain.SourceSurfaceFactory), SupportsImportProductCatalog: true,
	})
	templateRepo := newCatalogTestTemplateRepo()
	bindingRepo := newCatalogTestBindingRepo()
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "catalog", DocumentType: "import_product_catalog", Format: "zip",
		MappingRules: `{
			"version": 2, "mode": "header", "hasHeader": true,
			"columns": {"product.factory_sku": "SKU", "product.name": "Name"},
			"imageLayout": {
				"enabled": true,
				"coverDir": "covers",
				"namePattern": "{match}#{nn}"
			}
		}`,
	}
	_ = templateRepo.Create(context.Background(), tmpl)
	_ = bindingRepo.Create(context.Background(), &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: 1, DocumentType: "import_product_catalog", TemplateID: tmpl.ID, IsDefault: true,
	})
	mapping := NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)

	// ImageLayout is enabled but no asset store — attachCatalogImages must fail.
	uc := NewProductUseCase(masterRepo, nil, nil)
	uc = WithCatalogImportDeps(uc, mapping, profileRepo, nil)
	uc.(*productUseCase).catalog.extractRoot = t.TempDir()

	result, err := uc.ImportProductCatalog(context.Background(), dto.ImportProductCatalogInput{
		IntegrationProfileID: 1,
		ImportMode:           "skip_invalid",
		FilePath:             zipPath,
	})
	if err != nil {
		t.Fatalf("skip_invalid must not abort import: %v", err)
	}
	if result.SuccessCount != 0 || result.CreatedCount != 0 {
		t.Fatalf("attach failure must not count success: %+v", result)
	}
	if result.ErrorCount == 0 {
		t.Fatal("expected image attach error")
	}
	all, listErr := masterRepo.List(context.Background())
	if listErr != nil {
		t.Fatalf("list: %v", listErr)
	}
	if len(all) != 0 {
		t.Fatalf("must not persist row after attach failure: %+v", all)
	}
}

// Self-contained mocks so catalog-import tests compile without other _test.go files.

type catalogTestMasterRepo struct {
	mu     sync.Mutex
	byKey  map[string]*domain.ProductMaster
	lastID uint
}

func newCatalogTestMasterRepo() *catalogTestMasterRepo {
	return &catalogTestMasterRepo{byKey: make(map[string]*domain.ProductMaster)}
}

func catalogTestMasterKey(platform, sku string) string { return platform + "\x00" + sku }

func (m *catalogTestMasterRepo) Create(ctx context.Context, master *domain.ProductMaster) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastID++
	master.ID = m.lastID
	cp := *master
	m.byKey[catalogTestMasterKey(master.SupplierPlatform, master.FactorySKU)] = &cp
	return nil
}
func (m *catalogTestMasterRepo) FindByID(ctx context.Context, id uint) (*domain.ProductMaster, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *catalogTestMasterRepo) List(ctx context.Context) ([]domain.ProductMaster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.ProductMaster, 0, len(m.byKey))
	for _, master := range m.byKey {
		out = append(out, *master)
	}
	return out, nil
}
func (m *catalogTestMasterRepo) FindByPlatformAndSKU(ctx context.Context, platform, sku string) (*domain.ProductMaster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.byKey[catalogTestMasterKey(platform, sku)]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *p
	return &cp, nil
}
func (m *catalogTestMasterRepo) Update(ctx context.Context, master *domain.ProductMaster) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *master
	m.byKey[catalogTestMasterKey(master.SupplierPlatform, master.FactorySKU)] = &cp
	return nil
}

type catalogTestProfileRepo struct {
	profiles map[uint]*domain.IntegrationProfile
}

func newCatalogTestProfileRepo() *catalogTestProfileRepo {
	return &catalogTestProfileRepo{profiles: make(map[uint]*domain.IntegrationProfile)}
}
func (m *catalogTestProfileRepo) Create(ctx context.Context, p *domain.IntegrationProfile) error {
	if p.ID == 0 {
		p.ID = uint(len(m.profiles) + 1)
	}
	cp := *p
	m.profiles[p.ID] = &cp
	return nil
}
func (m *catalogTestProfileRepo) FindByID(ctx context.Context, id uint) (*domain.IntegrationProfile, error) {
	p, ok := m.profiles[id]
	if !ok {
		return nil, fmt.Errorf("profile %d not found", id)
	}
	cp := *p
	return &cp, nil
}
func (m *catalogTestProfileRepo) FindByProfileKey(ctx context.Context, key string) (*domain.IntegrationProfile, error) {
	for _, p := range m.profiles {
		if p.ProfileKey == key {
			cp := *p
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("profile key %q not found", key)
}
func (m *catalogTestProfileRepo) List(ctx context.Context) ([]domain.IntegrationProfile, error) {
	out := make([]domain.IntegrationProfile, 0, len(m.profiles))
	for _, p := range m.profiles {
		out = append(out, *p)
	}
	return out, nil
}
func (m *catalogTestProfileRepo) Update(ctx context.Context, p *domain.IntegrationProfile) error {
	if p == nil || p.ID == 0 {
		return fmt.Errorf("profile ID is required")
	}
	cp := *p
	m.profiles[p.ID] = &cp
	return nil
}
func (m *catalogTestProfileRepo) Delete(ctx context.Context, id uint) error {
	delete(m.profiles, id)
	return nil
}

type catalogTestTemplateRepo struct {
	mu      sync.Mutex
	records map[uint]*domain.DocumentTemplate
	byKey   map[string]*domain.DocumentTemplate
	lastID  uint
}

func newCatalogTestTemplateRepo() *catalogTestTemplateRepo {
	return &catalogTestTemplateRepo{
		records: make(map[uint]*domain.DocumentTemplate),
		byKey:   make(map[string]*domain.DocumentTemplate),
	}
}
func (m *catalogTestTemplateRepo) Create(ctx context.Context, t *domain.DocumentTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastID++
	t.ID = m.lastID
	t.CreatedAt = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t.UpdatedAt = t.CreatedAt
	cp := *t
	m.records[t.ID] = &cp
	m.byKey[t.TemplateKey] = &cp
	return nil
}
func (m *catalogTestTemplateRepo) FindByID(ctx context.Context, id uint) (*domain.DocumentTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.records[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}
func (m *catalogTestTemplateRepo) FindByKey(ctx context.Context, key string) (*domain.DocumentTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.byKey[key]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}
func (m *catalogTestTemplateRepo) List(ctx context.Context) ([]domain.DocumentTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.DocumentTemplate, 0, len(m.records))
	for _, t := range m.records {
		out = append(out, *t)
	}
	return out, nil
}
func (m *catalogTestTemplateRepo) ListByDocumentType(ctx context.Context, docType string) ([]domain.DocumentTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.DocumentTemplate
	for _, t := range m.records {
		if t.DocumentType == docType {
			out = append(out, *t)
		}
	}
	return out, nil
}
func (m *catalogTestTemplateRepo) Update(ctx context.Context, t *domain.DocumentTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *t
	m.records[t.ID] = &cp
	m.byKey[t.TemplateKey] = &cp
	return nil
}
func (m *catalogTestTemplateRepo) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.records, id)
	return nil
}

type catalogTestBindingRepo struct {
	mu      sync.Mutex
	records map[uint]*domain.IntegrationProfileTemplateBinding
	lastID  uint
}

func newCatalogTestBindingRepo() *catalogTestBindingRepo {
	return &catalogTestBindingRepo{records: make(map[uint]*domain.IntegrationProfileTemplateBinding)}
}
func (m *catalogTestBindingRepo) Create(ctx context.Context, b *domain.IntegrationProfileTemplateBinding) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastID++
	b.ID = m.lastID
	cp := *b
	m.records[b.ID] = &cp
	return nil
}
func (m *catalogTestBindingRepo) FindByID(ctx context.Context, id uint) (*domain.IntegrationProfileTemplateBinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.records[id]
	if !ok {
		return nil, nil
	}
	cp := *b
	return &cp, nil
}
func (m *catalogTestBindingRepo) ListByProfile(ctx context.Context, profileID uint) ([]domain.IntegrationProfileTemplateBinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.IntegrationProfileTemplateBinding
	for _, b := range m.records {
		if b.IntegrationProfileID == profileID {
			out = append(out, *b)
		}
	}
	return out, nil
}
func (m *catalogTestBindingRepo) ListByTemplateID(ctx context.Context, templateID uint) ([]domain.IntegrationProfileTemplateBinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.IntegrationProfileTemplateBinding
	for _, b := range m.records {
		if b.TemplateID == templateID {
			out = append(out, *b)
		}
	}
	return out, nil
}
func (m *catalogTestBindingRepo) FindDefaultByProfileAndType(ctx context.Context, profileID uint, docType string) (*domain.IntegrationProfileTemplateBinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, b := range m.records {
		if b.IntegrationProfileID == profileID && b.DocumentType == docType && b.IsDefault {
			cp := *b
			return &cp, nil
		}
	}
	return nil, nil
}
func (m *catalogTestBindingRepo) ClearDefaultByProfileAndType(ctx context.Context, profileID uint, docType string) error {
	return nil
}
func (m *catalogTestBindingRepo) Update(ctx context.Context, b *domain.IntegrationProfileTemplateBinding) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *b
	m.records[b.ID] = &cp
	return nil
}
func (m *catalogTestBindingRepo) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.records, id)
	return nil
}
func (m *catalogTestBindingRepo) CountByProfileID(ctx context.Context, profileID uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for _, b := range m.records {
		if b.IntegrationProfileID == profileID {
			n++
		}
	}
	return n, nil
}
