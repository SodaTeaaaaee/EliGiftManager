package app

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
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

	masterRepo := newMockProductMasterRepo()
	profileRepo := newMockIntegrationProfileRepoSimple()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "factory",
		FactorySupplierPlatform: "test-platform",
		ConnectorKey:            "factory-a",
	})
	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
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

	masterRepo := newMockProductMasterRepo()
	profileRepo := newMockIntegrationProfileRepoSimple()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "factory", ConnectorKey: "plat",
	})
	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
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
		path.Join(rootName, "a.csv"):                    csvBody,
		path.Join(rootName, "主图", "NestedWidget#01.png"): string(coverBytes),
		path.Join(rootName, "详情", "NestedWidget#02.png"): string(detailBytes),
	}); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	assetsRoot := t.TempDir()
	store := service.NewAssetStoreAt(assetsRoot)
	extractRoot := t.TempDir()

	masterRepo := newMockProductMasterRepo()
	profileRepo := newMockIntegrationProfileRepoSimple()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "factory",
		FactorySupplierPlatform: "test-platform",
		ConnectorKey:            "factory-a",
	})
	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
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
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	for name, body := range files {
		fw, err := w.Create(name)
		if err != nil {
			_ = w.Close()
			return err
		}
		if _, err := fw.Write([]byte(body)); err != nil {
			_ = w.Close()
			return err
		}
	}
	return w.Close()
}
