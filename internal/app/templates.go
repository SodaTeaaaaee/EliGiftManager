package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/alignment"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// Document types are a closed set. Each one fixes the template direction and
// the platform kind it belongs to; they ride on TemplateConfig.DocumentType.
const (
	DocumentTypeMembershipList = "membership_list"
	DocumentTypeOrderExport    = "order_export"
	DocumentTypeShipmentReturn = "shipment_return"
	DocumentTypeFactoryOrder   = "factory_order"
	DocumentTypeWriteback      = "writeback"
)

// DocumentTypeInfo is one row of the closed document-type catalog: the key
// stored on templates, the direction it locks, and the platform kind that may
// own such templates.
type DocumentTypeInfo struct {
	Key          string
	Direction    string
	PlatformKind string
}

// documentTypeCatalog is the single source of truth for the closed set.
var documentTypeCatalog = []DocumentTypeInfo{
	{Key: DocumentTypeMembershipList, Direction: string(domain.TemplateDirectionInput), PlatformKind: string(domain.PlatformKindSource)},
	{Key: DocumentTypeOrderExport, Direction: string(domain.TemplateDirectionInput), PlatformKind: string(domain.PlatformKindSource)},
	{Key: DocumentTypeShipmentReturn, Direction: string(domain.TemplateDirectionInput), PlatformKind: string(domain.PlatformKindFactory)},
	{Key: DocumentTypeFactoryOrder, Direction: string(domain.TemplateDirectionOutput), PlatformKind: string(domain.PlatformKindFactory)},
	{Key: DocumentTypeWriteback, Direction: string(domain.TemplateDirectionOutput), PlatformKind: string(domain.PlatformKindSource)},
}

// DocumentTypeCatalog returns the closed document-type set with the direction
// and platform kind each type locks.
func (ws *Workspace) DocumentTypeCatalog() []DocumentTypeInfo {
	return append([]DocumentTypeInfo(nil), documentTypeCatalog...)
}

func documentTypeInfo(key string) (DocumentTypeInfo, bool) {
	for _, info := range documentTypeCatalog {
		if info.Key == key {
			return info, true
		}
	}
	return DocumentTypeInfo{}, false
}

func factKindForDocumentType(documentType string) string {
	switch documentType {
	case DocumentTypeMembershipList, string(domain.InputFactKindMembership):
		return string(domain.InputFactKindMembership)
	case string(domain.InputFactKindOperatorGrant):
		return string(domain.InputFactKindOperatorGrant)
	default:
		return string(domain.InputFactKindRetailOrder)
	}
}

// validateTemplate checks the closed document-type set, the locked direction,
// the owning platform's kind, and that the side-relevant config parses:
// MappingJSON for input templates, LayoutJSON for output templates. Every read
// goes through the explicit store argument so callers control the transaction.
func validateTemplate(ctx context.Context, store domain.Store, t *domain.TemplateConfig) error {
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidTemplate)
	}
	info, ok := documentTypeInfo(t.DocumentType)
	if !ok {
		return fmt.Errorf("%w: document type %q is not one of membership_list, order_export, shipment_return, factory_order, writeback", ErrInvalidTemplate, t.DocumentType)
	}
	if t.Direction != info.Direction {
		return fmt.Errorf("%w: document type %s requires direction %s, got %q", ErrInvalidTemplate, info.Key, info.Direction, t.Direction)
	}
	platform, err := store.GetPlatform(ctx, t.PlatformID)
	if err != nil {
		if err == domain.ErrNotFound {
			return fmt.Errorf("%w: platform %d does not exist", ErrInvalidTemplate, t.PlatformID)
		}
		return err
	}
	if platform.Kind != info.PlatformKind {
		return fmt.Errorf("%w: document type %s belongs to %s platforms, but %q is a %s platform", ErrInvalidTemplate, info.Key, info.PlatformKind, platform.Name, platform.Kind)
	}
	switch info.Direction {
	case string(domain.TemplateDirectionInput):
		if _, err := alignment.ParseMappingConfig(t.MappingJSON); err != nil {
			return fmt.Errorf("%w: mapping: %w", ErrInvalidTemplate, err)
		}
	case string(domain.TemplateDirectionOutput):
		if strings.TrimSpace(t.LayoutJSON) == "" {
			return fmt.Errorf("%w: layout config is empty", ErrInvalidTemplate)
		}
		layout, err := alignment.ParseLayoutConfig(t.LayoutJSON)
		if err != nil {
			return fmt.Errorf("%w: layout: %w", ErrInvalidTemplate, err)
		}
		if len(layout.ColumnOrder) == 0 {
			return fmt.Errorf("%w: layout column order is empty", ErrInvalidTemplate)
		}
	}
	return nil
}

// CreateTemplate stores a new user template at version 1. Built-in rows are
// only ever written by EnsureBuiltinTemplates, so the flag is forced off here.
func (ws *Workspace) CreateTemplate(ctx context.Context, t *domain.TemplateConfig) error {
	t.ID = 0
	t.Builtin = false
	t.Version = 1
	if err := validateTemplate(ctx, ws.Store, t); err != nil {
		return err
	}
	return ws.Store.CreateTemplate(ctx, t)
}

func (ws *Workspace) GetTemplate(ctx context.Context, id uint) (*domain.TemplateConfig, error) {
	return ws.Store.GetTemplate(ctx, id)
}

// UpdateTemplate edits a user template in place: name, notes, and the config
// JSON are overwritten on the same row and Version advances so existing
// import/export snapshots (template id + version) stay distinguishable from
// later operations. Platform, document type, and direction are fixed at
// creation. Built-in rows are refused.
func (ws *Workspace) UpdateTemplate(ctx context.Context, t *domain.TemplateConfig) (*domain.TemplateConfig, error) {
	var updated *domain.TemplateConfig
	if err := ws.Store.WithTx(ctx, func(tx domain.Store) error {
		existing, err := tx.GetTemplate(ctx, t.ID)
		if err != nil {
			return err
		}
		if existing.Builtin {
			return fmt.Errorf("%w: template %d %q cannot be updated; copy it into an active template instead", ErrBuiltinTemplate, existing.ID, existing.Name)
		}
		if t.PlatformID != existing.PlatformID || t.DocumentType != existing.DocumentType || t.Direction != existing.Direction {
			return fmt.Errorf("%w: platform, document type, and direction are fixed after creation", ErrInvalidTemplate)
		}
		next := *existing
		next.Name = t.Name
		next.Notes = t.Notes
		next.MappingJSON = t.MappingJSON
		next.LayoutJSON = t.LayoutJSON
		next.ExtraData = t.ExtraData
		next.Version = existing.Version + 1
		if err := validateTemplate(ctx, tx, &next); err != nil {
			return err
		}
		if err := tx.UpdateTemplate(ctx, &next); err != nil {
			return err
		}
		updated = &next
		return nil
	}); err != nil {
		return nil, err
	}
	return updated, nil
}

// DeleteTemplate removes a user template. Historical snapshots keep the
// dangling id and version; nothing cascades. Built-in rows are refused.
func (ws *Workspace) DeleteTemplate(ctx context.Context, id uint) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		existing, err := tx.GetTemplate(ctx, id)
		if err != nil {
			return err
		}
		if existing.Builtin {
			return fmt.Errorf("%w: template %d %q cannot be deleted", ErrBuiltinTemplate, existing.ID, existing.Name)
		}
		return tx.DeleteTemplate(ctx, id)
	})
}

func (ws *Workspace) ListTemplates(ctx context.Context) ([]domain.TemplateConfig, error) {
	return ws.Store.ListTemplates(ctx)
}

// findActiveTemplate returns the active (non-builtin) template for a platform,
// direction, and document type, or nil when none exists. Among several
// candidates the most recently updated wins, ties going to the greatest id.
// Every read goes through the explicit store argument so callers control which
// transaction or connection the lookup joins.
func findActiveTemplate(ctx context.Context, store domain.Store, platformID uint, direction domain.TemplateDirection, documentType string) (*domain.TemplateConfig, error) {
	templates, err := store.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	var best *domain.TemplateConfig
	for i := range templates {
		t := &templates[i]
		if t.Builtin || t.PlatformID != platformID || t.Direction != string(direction) || t.DocumentType != documentType {
			continue
		}
		if best == nil || t.UpdatedAt.After(best.UpdatedAt) || (t.UpdatedAt.Equal(best.UpdatedAt) && t.ID > best.ID) {
			best = t
		}
	}
	return best, nil
}

// requireActiveTemplate is findActiveTemplate for operations that cannot run
// without a template: a missing one is ErrNoActiveTemplate naming the platform
// and document type instead of a silent built-in default.
func requireActiveTemplate(ctx context.Context, store domain.Store, platformID uint, direction domain.TemplateDirection, documentType string) (*domain.TemplateConfig, error) {
	tpl, err := findActiveTemplate(ctx, store, platformID, direction, documentType)
	if err != nil {
		return nil, err
	}
	if tpl != nil {
		return tpl, nil
	}
	name := fmt.Sprintf("#%d", platformID)
	if p, err := store.GetPlatform(ctx, platformID); err == nil {
		name = fmt.Sprintf("%q (%s)", p.Name, p.Key)
	}
	return nil, fmt.Errorf("%w: platform %s has no active %s/%s template; copy the built-in one or create a template first", ErrNoActiveTemplate, name, documentType, direction)
}

// checkUsableTemplate verifies a template picked by the caller for a concrete
// operation: it must belong to the platform, be an active (non-builtin) row,
// and carry the expected direction and, when given, document type.
func checkUsableTemplate(tpl *domain.TemplateConfig, platformID uint, direction domain.TemplateDirection, documentType string) error {
	if tpl.PlatformID != platformID {
		return fmt.Errorf("%w: template %d %q does not belong to platform %d", ErrTemplateMismatch, tpl.ID, tpl.Name, platformID)
	}
	if tpl.Builtin {
		return fmt.Errorf("%w: template %d %q cannot be used directly; copy it into an active template first", ErrBuiltinTemplate, tpl.ID, tpl.Name)
	}
	if tpl.Direction != string(direction) {
		return fmt.Errorf("%w: template %d %q is an %s template, this operation needs %s", ErrTemplateMismatch, tpl.ID, tpl.Name, tpl.Direction, direction)
	}
	if documentType != "" && tpl.DocumentType != documentType {
		return fmt.Errorf("%w: template %d %q is a %s template, this operation needs %s", ErrTemplateMismatch, tpl.ID, tpl.Name, tpl.DocumentType, documentType)
	}
	if _, ok := documentTypeInfo(tpl.DocumentType); !ok {
		return fmt.Errorf("%w: template %d %q has unknown document type %q", ErrTemplateMismatch, tpl.ID, tpl.Name, tpl.DocumentType)
	}
	return nil
}
