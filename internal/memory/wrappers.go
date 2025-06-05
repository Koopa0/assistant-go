package memory

import (
	"context"
	"fmt"

	coremem "github.com/koopa0/assistant-go/internal/core/memory"
)

// dbMemoryWrapper adapts database operations to specific memory types
type dbMemoryWrapper struct {
	store   *Store
	memType coremem.Type
}

func (d *dbMemoryWrapper) Store(ctx context.Context, entry coremem.Entry) error {
	// Force the entry type to match this wrapper's type
	entry.Type = d.memType
	return d.store.Store(ctx, entry)
}

func (d *dbMemoryWrapper) Retrieve(ctx context.Context, id string) (*coremem.Entry, error) {
	entry, err := d.store.Retrieve(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify the entry type matches
	if entry.Type != d.memType {
		return nil, fmt.Errorf("entry type mismatch: expected %s, got %s", d.memType, entry.Type)
	}

	return entry, nil
}

func (d *dbMemoryWrapper) Search(ctx context.Context, criteria coremem.SearchCriteria) ([]coremem.Entry, error) {
	// Filter criteria to only this memory type
	criteria.Types = []coremem.Type{d.memType}
	return d.store.Search(ctx, criteria)
}

func (d *dbMemoryWrapper) Delete(ctx context.Context, id string) error {
	// TODO: Add type verification before deletion
	return d.store.Delete(ctx, id)
}
