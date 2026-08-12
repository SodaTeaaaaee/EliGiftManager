package app

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/tabular"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
)

// catalogImportDeps holds optional collaborators used only by ImportProductCatalog.
// Constructed lazily so the existing NewProductUseCase signature stays stable.
type catalogImportDeps struct {
	templateMapping *TemplateMappingService
	profileRepo     domain.IntegrationProfileRepository
	assetStore      *service.AssetStore
	evidence        *ImportEvidenceUseCase
	// extractRoot overrides data/tmp/catalog-import parent (tests only).
	extractRoot string
}

func WithCatalogImportEvidence(uc ProductUseCase, evidence *ImportEvidenceUseCase) ProductUseCase {
	p, ok := uc.(*productUseCase)
	if ok && p.catalog != nil {
		p.catalog.evidence = evidence
	}
	return uc
}

// WithCatalogImportDeps attaches template/profile/asset collaborators for catalog import.
// Call after NewProductUseCase when the controller needs ImportProductCatalog.
// assetStore may be nil when image association is not required.
func WithCatalogImportDeps(
	uc ProductUseCase,
	mapping *TemplateMappingService,
	profileRepo domain.IntegrationProfileRepository,
	assetStore *service.AssetStore,
) ProductUseCase {
	p, ok := uc.(*productUseCase)
	if !ok {
		return uc
	}
	p.catalog = &catalogImportDeps{
		templateMapping: mapping,
		profileRepo:     profileRepo,
		assetStore:      assetStore,
	}
	return p
}

// ImportProductCatalog upserts ProductMaster rows from a template-mapped sheet
// (document type import_product_catalog, product.* dest keys).
//
// Unique key: (supplier_platform, factory_sku). Platform resolution order:
//  1. product.supplier_platform from the mapped row
//  2. profile.FactorySupplierPlatform, else profile.ConnectorKey
//  3. empty string is rejected
//
// When FilePath is a .zip, the archive is extracted under data/tmp/catalog-import/<uuid>/
// and cleaned up on return. Tabular file selection uses ImageLayout.TabularGlob (default *.csv).
// When ImageLayout.Enabled, cover/detail images are matched by NamePattern and stored via AssetStore.
func (uc *productUseCase) ImportProductCatalog(ctx context.Context, input dto.ImportProductCatalogInput) (*dto.ImportProductCatalogResult, error) {
	if uc.catalog == nil || uc.catalog.templateMapping == nil || uc.catalog.profileRepo == nil {
		return nil, fmt.Errorf("import product catalog: catalog import deps not configured")
	}
	mode := input.ImportMode
	if mode == "" {
		mode = "skip_invalid"
	}
	if mode != "reject_all" && mode != "skip_invalid" {
		return nil, fmt.Errorf("invalid importMode %q: must be \"reject_all\" or \"skip_invalid\"", mode)
	}
	if input.IntegrationProfileID == 0 {
		return nil, fmt.Errorf("integrationProfileId is required")
	}

	profile, err := uc.catalog.profileRepo.FindByID(ctx, input.IntegrationProfileID)
	if err != nil {
		return nil, fmt.Errorf("integration profile %d not found: %w", input.IntegrationProfileID, err)
	}

	// Platform fallback when the row does not carry product.supplier_platform.
	// Prefer FactorySupplierPlatform (group B); fall back to ConnectorKey for legacy profiles.
	defaultPlatform := strings.TrimSpace(profile.FactorySupplierPlatform)
	if defaultPlatform == "" {
		defaultPlatform = strings.TrimSpace(profile.ConnectorKey)
	}

	tmpl, rules, err := uc.catalog.templateMapping.ResolveTemplateAndRules(ctx, profile.ID, "import_product_catalog")
	if err != nil {
		return nil, fmt.Errorf("template pipeline: %w", err)
	}
	_ = tmpl

	orderedRows, headers, headerRows, total, imageRoot, cleanup, err := loadImportRows(input.FilePath, input.Rows, rules, uc.catalog)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return nil, err
	}
	var evidenceRun *domain.ImportRun
	var evidenceRecords []domain.ImportRawRecord
	if uc.catalog.evidence != nil {
		evidenceRows, unmapped := importEvidenceRows(orderedRows, headers, headerRows)
		assets := make([][]map[string]string, len(evidenceRows))
		if strings.EqualFold(filepath.Ext(input.FilePath), ".zip") && len(assets) > 0 {
			metadata, metadataErr := zipAssetMetadata(input.FilePath)
			if metadataErr != nil {
				return nil, metadataErr
			}
			assets[0] = metadata
		}
		parserMetadata := fmt.Sprintf(`{"hasHeader":%t,"sheetName":%q}`, rules.HasHeader, rules.SheetName)
		evidenceRun, evidenceRecords, err = uc.catalog.evidence.StartImportEvidence(ctx, "product_catalog", input.IntegrationProfileID, mode, input.FilePath, parserMetadata, evidenceRows, unmapped, assets)
		if err != nil {
			return nil, fmt.Errorf("start catalog import evidence: %w", err)
		}
	}

	type pendingMaster struct {
		idx    int
		master *domain.ProductMaster
	}
	var pending []pendingMaster
	var rowErrors []dto.ImportProductCatalogError
	var rowWarnings rowWarningCollector

	for i := 0; i < total; i++ {
		var applied map[string]string
		var mapErr error
		var warnings []string
		if len(orderedRows) > 0 {
			applied, warnings, mapErr = ApplyRow(orderedRows[i], headers, rules)
		} else {
			applied, warnings, mapErr = applyHeaderMap(headerRows[i], rules)
		}
		rowWarnings.add(i, warnings)
		if mapErr != nil {
			markImportEvidenceFailure(evidenceRecords, i, "mapping_error", mapErr.Error(), warnings)
			rowErrors = append(rowErrors, dto.ImportProductCatalogError{RowIndex: i, Reason: mapErr.Error()})
			if mode == "reject_all" {
				break
			}
			continue
		}

		master, buildErr := buildProductMasterFromApplied(applied, defaultPlatform)
		if buildErr != nil {
			markImportEvidenceFailure(evidenceRecords, i, "validation_error", buildErr.Error(), warnings)
			rowErrors = append(rowErrors, dto.ImportProductCatalogError{RowIndex: i, Reason: buildErr.Error()})
			if mode == "reject_all" {
				break
			}
			continue
		}

		pending = append(pending, pendingMaster{idx: i, master: master})
	}

	result := &dto.ImportProductCatalogResult{
		ImportRunID:      importEvidenceRunID(evidenceRun),
		EvidenceDisabled: uc.catalog.evidence != nil && evidenceRun == nil,
		TotalProcessed:   total,
		ErrorCount:       len(rowErrors),
		Errors:           rowErrors,
		Warnings:         rowWarnings.warnings(),
	}
	if mode == "reject_all" && len(rowErrors) > 0 {
		result.SuccessCount = 0
		if evidenceRun != nil {
			if err := uc.catalog.evidence.CompleteImportEvidence(ctx, evidenceRun, evidenceRecords, "rejected"); err != nil {
				return nil, err
			}
		}
		return result, nil
	}

	now := time.Now()
	for _, p := range pending {
		// Image files are only materialised after reject_all has passed the full
		// validation gate. The controller supplies a staging store for reject_all.
		if rules.ImageLayout != nil && rules.ImageLayout.Enabled {
			if attachErr := attachCatalogImages(p.master, rules.ImageLayout, imageRoot, uc.catalog.assetStore); attachErr != nil {
				if mode == "reject_all" {
					return nil, fmt.Errorf("attach catalog images for row %d: %w", p.idx, attachErr)
				}
				result.Errors = append(result.Errors, dto.ImportProductCatalogError{
					RowIndex: p.idx, Reason: fmt.Sprintf("images: %v", attachErr),
				})
				result.ErrorCount++
			}
		}
		existing, findErr := uc.masterRepo.FindByPlatformAndSKU(ctx, p.master.SupplierPlatform, p.master.FactorySKU)
		if findErr == nil && existing != nil {
			// Upsert: overwrite mutable fields, keep ID / timestamps base.
			existing.Name = p.master.Name
			if p.master.SupplierProductRef != "" {
				existing.SupplierProductRef = p.master.SupplierProductRef
			}
			if p.master.ProductKind != "" {
				existing.ProductKind = p.master.ProductKind
			}
			if p.master.ExtraData != "" {
				existing.ExtraData = p.master.ExtraData
			}
			// Image paths: replace when the import resolved any (cover or details).
			if p.master.CoverImagePath != "" || p.master.DetailImagePaths != "" {
				existing.CoverImagePath = p.master.CoverImagePath
				existing.DetailImagePaths = p.master.DetailImagePaths
			}
			existing.UpdatedAt = now
			if err := uc.masterRepo.Update(ctx, existing); err != nil {
				if mode == "reject_all" {
					return nil, fmt.Errorf("update product catalog row %d: %w", p.idx, err)
				}
				result.Errors = append(result.Errors, dto.ImportProductCatalogError{
					RowIndex: p.idx, Reason: fmt.Sprintf("update: %v", err),
				})
				result.ErrorCount++
				continue
			}
			result.UpdatedCount++
			result.SuccessCount++
			result.Masters = append(result.Masters, productMasterToDTO(existing))
			markImportEvidenceSuccess(evidenceRecords, p.idx, "product_master", existing.ID)
			continue
		}

		p.master.CreatedAt = now
		p.master.UpdatedAt = now
		if err := uc.masterRepo.Create(ctx, p.master); err != nil {
			if mode == "reject_all" {
				return nil, fmt.Errorf("create product catalog row %d: %w", p.idx, err)
			}
			result.Errors = append(result.Errors, dto.ImportProductCatalogError{
				RowIndex: p.idx, Reason: fmt.Sprintf("create: %v", err),
			})
			result.ErrorCount++
			continue
		}
		result.CreatedCount++
		result.SuccessCount++
		result.Masters = append(result.Masters, productMasterToDTO(p.master))
		markImportEvidenceSuccess(evidenceRecords, p.idx, "product_master", p.master.ID)
	}
	if evidenceRun != nil {
		status := "completed"
		if result.ErrorCount > 0 {
			status = "partial_success"
		}
		if err := uc.catalog.evidence.CompleteImportEvidence(ctx, evidenceRun, evidenceRecords, status); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func buildProductMasterFromApplied(applied map[string]string, defaultPlatform string) (*domain.ProductMaster, error) {
	get := func(field string) string {
		if v, ok := applied["product."+field]; ok {
			return strings.TrimSpace(v)
		}
		// Unprefixed fallback for simple templates.
		return strings.TrimSpace(applied[field])
	}

	platform := get("supplier_platform")
	if platform == "" {
		platform = get("platform")
	}
	if platform == "" {
		platform = defaultPlatform
	}
	sku := get("factory_sku")
	if sku == "" {
		sku = get("sku")
	}
	name := get("name")
	if platform == "" {
		return nil, fmt.Errorf("product.supplier_platform is required (and profile has no ConnectorKey fallback)")
	}
	if sku == "" {
		return nil, fmt.Errorf("product.factory_sku is required")
	}
	if name == "" {
		// Default name to SKU so a minimal catalog sheet still imports.
		name = sku
	}
	kind := get("product_kind")
	if kind == "" {
		kind = "other"
	}
	if !validProductKinds[kind] {
		return nil, fmt.Errorf("invalid product_kind %q", kind)
	}

	return &domain.ProductMaster{
		SupplierPlatform:   platform,
		FactorySKU:         sku,
		SupplierProductRef: get("supplier_product_ref"),
		Name:               name,
		ProductKind:        domain.ProductKind(kind),
		ExtraData:          get("extra_data"),
	}, nil
}

// loadImportRows normalises FilePath vs pre-parsed Rows into either ordered cells
// (preferred) or header-keyed maps. Zip archives are extracted; imageRoot is the
// catalog content root (or the tabular file's parent for non-zip). cleanup removes
// the extract directory when non-nil.
func loadImportRows(
	filePath string,
	rows []map[string]string,
	rules *TemplateMappingRules,
	deps *catalogImportDeps,
) (ordered [][]string, headers []string, headerRows []map[string]string, total int, imageRoot string, cleanup func(), err error) {
	if filePath != "" {
		ext := strings.ToLower(filepath.Ext(filePath))
		tabularPath := filePath
		if ext == ".zip" {
			extractDir, extractErr := extractCatalogZip(filePath, deps)
			if extractErr != nil {
				return nil, nil, nil, 0, "", nil, extractErr
			}
			cleanup = func() { _ = os.RemoveAll(extractDir) }

			// Platform exports often wrap CSV + image dirs under a single parent
			// folder (e.g. "从工厂平台导出-商品列表/主图/..."). Resolve that content root
			// so CoverDir/DetailDir remain simple relative names like "主图".
			imageRoot = resolveCatalogContentRoot(extractDir, rules.ImageLayout)

			glob := "*.csv"
			if rules.ImageLayout != nil && strings.TrimSpace(rules.ImageLayout.TabularGlob) != "" {
				glob = rules.ImageLayout.TabularGlob
			}
			// Prefer content root; fall back to full extract tree (any depth).
			found, findErr := findTabularInDir(imageRoot, glob)
			if findErr != nil && imageRoot != extractDir {
				found, findErr = findTabularInDir(extractDir, glob)
			}
			if findErr != nil {
				return nil, nil, nil, 0, "", cleanup, findErr
			}
			tabularPath = found
		} else {
			imageRoot = filepath.Dir(filePath)
		}

		sheet, readErr := tabular.ReadTabularFile(tabularPath, tabular.ReadOptions{
			HasHeader: rules.HasHeader,
			Encoding:  "auto",
			SheetName: rules.SheetName,
		})
		if readErr != nil {
			return nil, nil, nil, 0, "", cleanup, fmt.Errorf("read tabular file: %w", readErr)
		}
		return sheet.Rows, sheet.Headers, nil, len(sheet.Rows), imageRoot, cleanup, nil
	}
	if len(rows) == 0 {
		return nil, nil, nil, 0, "", nil, fmt.Errorf("no rows and no filePath provided")
	}
	return nil, nil, rows, len(rows), "", nil, nil
}

// resolveCatalogContentRoot finds the logical content root inside an extracted
// catalog zip. Many platform exports nest everything under a single top-level
// folder (e.g. "从工厂平台导出-商品列表/主图/xxx#01.png" + ".../a.csv") while
// ImageLayout.CoverDir is just "主图".
//
// Resolution order:
//  1. CoverDir (or DetailDir) present directly under extractDir → extractDir
//  2. Exactly one child directory that contains the marker dir → that child
//  3. Any direct child containing the marker (lexicographically first)
//  4. Recursive walk: parent of the first directory named like the marker
//  5. Exactly one child directory (no marker configured / not found) → that child
//  6. Otherwise extractDir
func resolveCatalogContentRoot(extractDir string, layout *CatalogImageLayout) string {
	marker := ""
	if layout != nil {
		marker = strings.TrimSpace(layout.CoverDir)
		if marker == "" {
			marker = strings.TrimSpace(layout.DetailDir)
		}
	}
	marker = filepath.Clean(filepath.FromSlash(marker))
	if marker == "." || marker == "" {
		marker = ""
	}

	if marker != "" {
		// 1. Direct hit at extract root.
		if dirExists(filepath.Join(extractDir, marker)) {
			return extractDir
		}

		// 2–3. One-level child lookup: */marker
		children, err := listImmediateSubdirs(extractDir)
		if err == nil {
			var hits []string
			for _, child := range children {
				if dirExists(filepath.Join(child, marker)) {
					hits = append(hits, child)
				}
			}
			if len(hits) == 1 {
				return hits[0]
			}
			if len(hits) > 1 {
				sort.Strings(hits)
				return hits[0]
			}
		}

		// 4. Recursive: first directory whose base name equals marker → its parent.
		if parent := findNamedDirParent(extractDir, filepath.Base(marker)); parent != "" {
			return parent
		}
	}

	// 5. Unique wrapper folder heuristic (common even without image dirs).
	if children, err := listImmediateSubdirs(extractDir); err == nil && len(children) == 1 {
		// Only promote when the sole child looks like a content wrapper
		// (contains at least one file or subdir — always true after extract).
		return children[0]
	}
	return extractDir
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func listImmediateSubdirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, filepath.Join(root, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

// findNamedDirParent walks root and returns the parent path of the first
// directory whose base name equals dirName (case-sensitive, OS-native).
func findNamedDirParent(root, dirName string) string {
	if dirName == "" || dirName == "." {
		return ""
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || !d.IsDir() {
			return nil
		}
		if path == root {
			return nil
		}
		if d.Name() == dirName {
			found = filepath.Dir(path)
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func applyHeaderMap(row map[string]string, rules *TemplateMappingRules) (map[string]string, []string, error) {
	if rules.Mode == "positional" {
		return nil, nil, fmt.Errorf("positional mapping requires ordered row cells from a file path")
	}
	headers := make([]string, 0, len(row))
	cells := make([]string, 0, len(row))
	for k, v := range row {
		headers = append(headers, k)
		cells = append(cells, v)
	}
	return ApplyRow(cells, headers, rules)
}

// extractCatalogZip unpacks zipPath into data/tmp/catalog-import/<uuid>/ (or deps.extractRoot).
func extractCatalogZip(zipPath string, deps *catalogImportDeps) (string, error) {
	base := ""
	if deps != nil {
		base = deps.extractRoot
	}
	if base == "" {
		dataDir, err := service.ResolveDataDir()
		if err != nil {
			return "", fmt.Errorf("catalog zip extract: resolve data dir: %w", err)
		}
		base = filepath.Join(dataDir, "tmp", "catalog-import")
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", fmt.Errorf("catalog zip extract: mkdir base: %w", err)
	}
	// Unique subdir under catalog-import/ (uuid-like temp name).
	extractDir, err := os.MkdirTemp(base, "")
	if err != nil {
		return "", fmt.Errorf("catalog zip extract: mkdir extract: %w", err)
	}

	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		_ = os.RemoveAll(extractDir)
		return "", fmt.Errorf("catalog zip extract: open %q: %w", zipPath, err)
	}
	defer archive.Close()

	cleanExtract, err := filepath.Abs(extractDir)
	if err != nil {
		_ = os.RemoveAll(extractDir)
		return "", fmt.Errorf("catalog zip extract: abs extract: %w", err)
	}

	for _, f := range archive.File {
		// Normalize zip entry names (forward slashes) to local paths.
		name := filepath.FromSlash(f.Name)
		destPath := filepath.Join(extractDir, name)
		cleanDest, absErr := filepath.Abs(destPath)
		if absErr != nil {
			continue
		}
		// Zip-slip guard.
		if cleanDest != cleanExtract && !strings.HasPrefix(cleanDest, cleanExtract+string(os.PathSeparator)) {
			continue
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(destPath, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			continue
		}
		rc, openErr := f.Open()
		if openErr != nil {
			continue
		}
		out, createErr := os.Create(destPath)
		if createErr != nil {
			rc.Close()
			continue
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if copyErr != nil {
			_ = os.Remove(destPath)
		}
	}
	return extractDir, nil
}

// findTabularInDir selects a tabular file under root matching glob.
// Multiple hits: lexicographically first (by full path). Falls back to recursive
// basename match when top-level glob yields nothing.
func findTabularInDir(root, glob string) (string, error) {
	glob = strings.TrimSpace(glob)
	if glob == "" {
		glob = "*.csv"
	}

	var matches []string
	if direct, err := filepath.Glob(filepath.Join(root, glob)); err == nil {
		matches = append(matches, direct...)
	}
	if len(matches) == 0 {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil
			}
			ok, matchErr := filepath.Match(glob, filepath.Base(path))
			if matchErr == nil && ok {
				matches = append(matches, path)
			}
			return nil
		})
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no tabular file matching %q found in archive", glob)
	}
	sort.Strings(matches)
	return matches[0], nil
}

// attachCatalogImages resolves cover + detail images for one master and stores them.
// Soft no-op when imageRoot is empty or layout has no dirs; hard error when store is required but nil.
func attachCatalogImages(
	master *domain.ProductMaster,
	layout *CatalogImageLayout,
	imageRoot string,
	store *service.AssetStore,
) error {
	if layout == nil || !layout.Enabled {
		return nil
	}
	if imageRoot == "" {
		return nil
	}
	if store == nil {
		return fmt.Errorf("asset store is not configured")
	}

	matchValue := catalogMatchValue(master, layout.MatchField)
	if matchValue == "" {
		return nil
	}

	namePattern := layout.NamePattern
	if namePattern == "" {
		namePattern = "{match}#{nn}"
	}
	exts := layout.ImageExts
	if len(exts) == 0 {
		exts = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}
	}

	coverPick := strings.ToLower(strings.TrimSpace(layout.CoverPick))
	if coverPick == "" {
		coverPick = "lowest_nn"
	}
	if coverPick != "lowest_nn" {
		return fmt.Errorf("unsupported coverPick %q (only lowest_nn)", coverPick)
	}

	var coverRel string
	if strings.TrimSpace(layout.CoverDir) != "" {
		coverDir := filepath.Join(imageRoot, filepath.FromSlash(layout.CoverDir))
		candidates, err := listPatternImages(coverDir, matchValue, namePattern, exts)
		if err != nil {
			return err
		}
		if len(candidates) > 0 {
			// lowest_nn — candidates already sorted ascending by nn.
			rel, storeErr := store.StoreFile(candidates[0].path)
			if storeErr != nil {
				return fmt.Errorf("store cover: %w", storeErr)
			}
			coverRel = rel
		}
	}

	var detailRels []string
	if strings.TrimSpace(layout.DetailDir) != "" {
		detailDir := filepath.Join(imageRoot, filepath.FromSlash(layout.DetailDir))
		candidates, err := listPatternImages(detailDir, matchValue, namePattern, exts)
		if err != nil {
			return err
		}
		// When cover and detail share a directory, skip the lowest_nn file already
		// chosen as cover so it is not duplicated in DetailImagePaths.
		sharedDir := coverRel != "" &&
			filepath.Clean(filepath.FromSlash(layout.CoverDir)) == filepath.Clean(filepath.FromSlash(layout.DetailDir))
		for i, c := range candidates {
			if sharedDir && i == 0 {
				continue
			}
			rel, storeErr := store.StoreFile(c.path)
			if storeErr != nil {
				return fmt.Errorf("store detail: %w", storeErr)
			}
			detailRels = append(detailRels, rel)
		}
	}

	if coverRel != "" {
		master.CoverImagePath = coverRel
	}
	if len(detailRels) > 0 {
		raw, err := json.Marshal(detailRels)
		if err != nil {
			return fmt.Errorf("marshal detail paths: %w", err)
		}
		master.DetailImagePaths = string(raw)
	}
	return nil
}

type imageCandidate struct {
	path string
	nn   int
}

// listPatternImages lists image files in dir whose stem matches namePattern with
// the concrete match value. Results are sorted by nn ascending (then path).
func listPatternImages(dir, matchValue, namePattern string, exts []string) ([]imageCandidate, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat image dir %q: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	re, err := namePatternRegex(namePattern, matchValue)
	if err != nil {
		return nil, err
	}
	extSet := make(map[string]struct{}, len(exts))
	for _, e := range exts {
		extSet[strings.ToLower(e)] = struct{}{}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read image dir %q: %w", dir, err)
	}

	var out []imageCandidate
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if _, ok := extSet[ext]; !ok {
			continue
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		m := re.FindStringSubmatch(stem)
		if m == nil {
			continue
		}
		nn := 0
		if len(m) > 1 {
			nn, _ = strconv.Atoi(m[1])
		}
		out = append(out, imageCandidate{path: filepath.Join(dir, name), nn: nn})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].nn != out[j].nn {
			return out[i].nn < out[j].nn
		}
		return out[i].path < out[j].path
	})
	return out, nil
}

// namePatternRegex builds a stem regex from a pattern like "{match}#{nn}".
// The first capture group is the nn digits.
func namePatternRegex(pattern, matchValue string) (*regexp.Regexp, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		pattern = "{match}#{nn}"
	}
	var b strings.Builder
	b.WriteString("^")
	remaining := pattern
	nnGroups := 0
	for {
		iMatch := strings.Index(remaining, "{match}")
		iNN := strings.Index(remaining, "{nn}")
		if iMatch < 0 && iNN < 0 {
			b.WriteString(regexp.QuoteMeta(remaining))
			break
		}
		useMatch := iMatch >= 0 && (iNN < 0 || iMatch < iNN)
		if useMatch {
			b.WriteString(regexp.QuoteMeta(remaining[:iMatch]))
			b.WriteString(regexp.QuoteMeta(matchValue))
			remaining = remaining[iMatch+len("{match}"):]
			continue
		}
		b.WriteString(regexp.QuoteMeta(remaining[:iNN]))
		b.WriteString(`(\d+)`)
		nnGroups++
		remaining = remaining[iNN+len("{nn}"):]
	}
	b.WriteString("$")
	if nnGroups == 0 {
		// Pattern without {nn}: still allow exact match (nn treated as 0).
		return regexp.Compile(b.String())
	}
	return regexp.Compile(b.String())
}

func catalogMatchValue(master *domain.ProductMaster, field string) string {
	field = strings.TrimSpace(field)
	switch field {
	case "", "product.name", "name":
		return master.Name
	case "product.factory_sku", "factory_sku", "sku":
		return master.FactorySKU
	case "product.supplier_product_ref", "supplier_product_ref":
		return master.SupplierProductRef
	case "product.supplier_platform", "supplier_platform", "platform":
		return master.SupplierPlatform
	default:
		// Unknown field: try stripping product. prefix for name-like fallback.
		if strings.HasPrefix(field, "product.") {
			return catalogMatchValue(master, strings.TrimPrefix(field, "product."))
		}
		return ""
	}
}
