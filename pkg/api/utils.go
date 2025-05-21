package typesenseapi

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	pkgx "github.com/foomo/typesense/pkg"
	"github.com/typesense/typesense-go/v3/typesense/api"
	"github.com/typesense/typesense-go/v3/typesense/api/pointer"
	"go.uber.org/zap"
)

const defaultSearchPresetName = "default"

// buildSearchParams will return the search collection parameters
func buildSearchParams(
	params *pkgx.SearchParameters,
) *api.SearchCollectionParams {
	if params.Page < 1 {
		params.Page = 1
	}

	searchParams := &api.SearchCollectionParams{
		Page: pointer.Int(params.Page),
	}

	if params.PresetName != "" {
		searchParams.Preset = pointer.String(params.PresetName)
	} else {
		searchParams.Preset = pointer.String(defaultSearchPresetName)
	}

	if params.Query != "" {
		searchParams.Q = pointer.String(params.Query)
	}

	if params.Modify != nil {
		params.Modify(searchParams)
	}

	return searchParams
}

func (b *BaseAPI[indexDocument, returnType]) generateRevisionID() pkgx.RevisionID {
	return pkgx.RevisionID(time.Now().Format("2006-01-02-15-04")) // "YYYY-MM-DD-HH-MM"
}

func formatCollectionName(indexID pkgx.IndexID, revisionID pkgx.RevisionID) string {
	return fmt.Sprintf("%s-%s", indexID, revisionID)
}

func extractRevisionID(collectionName, name string) pkgx.RevisionID {
	if !strings.HasPrefix(collectionName, name+"-") {
		return ""
	}

	revisionID := strings.TrimPrefix(collectionName, name+"-")

	// Validate that the extracted revision ID follows YYYY-MM-DD-HH-MM format (16 chars)
	if len(revisionID) != 16 {
		return ""
	}

	return pkgx.RevisionID(revisionID)
}

// ensureAliasMapping ensures an alias correctly points to the specified collection.
func (b *BaseAPI[indexDocument, returnType]) ensureAliasMapping(ctx context.Context, indexID pkgx.IndexID, collectionName string) error {
	_, err := b.client.Aliases().Upsert(ctx, string(indexID), &api.CollectionAliasSchema{
		CollectionName: collectionName,
	})
	if err != nil {
		b.l.Error("failed to upsert alias",
			zap.String("alias", string(indexID)),
			zap.String("collection", collectionName),
			zap.Error(err),
		)
	}
	return err
}

func (b *BaseAPI[indexDocument, returnType]) pruneOldCollections(ctx context.Context, alias, currentCollection string) error {
	// Step 1: Retrieve all collections
	collections, err := b.client.Collections().Retrieve(ctx)
	if err != nil {
		b.l.Error("failed to retrieve collections", zap.Error(err))
		return err
	}

	var oldCollections []string
	for _, col := range collections {
		if strings.HasPrefix(col.Name, alias+"-") && col.Name != currentCollection {
			oldCollections = append(oldCollections, col.Name)
		}
	}

	// Step 2: Sort collections by timestamp (latest first)
	sort.Slice(oldCollections, func(i, j int) bool {
		return oldCollections[i] > oldCollections[j] // Reverse order
	})

	// Step 3: Delete all but the latest two collections
	if len(oldCollections) > 1 {
		toDelete := oldCollections[1:] // Keep only the latest two
		for _, col := range toDelete {
			_, err := b.client.Collection(col).Delete(ctx)
			if err != nil {
				b.l.Error("failed to delete collection", zap.String("collection", col), zap.Error(err))
			} else {
				b.l.Info("deleted old collection", zap.String("collection", col))
			}
		}
	}

	return nil
}

// fetchExistingCollections retrieves all existing collections and stores them in a map for quick lookup.
func (b *BaseAPI[indexDocument, returnType]) fetchExistingCollections(ctx context.Context) (map[string]bool, error) {
	collections, err := b.client.Collections().Retrieve(ctx)
	if err != nil {
		b.l.Error("failed to retrieve collections", zap.Error(err))
		return nil, err
	}

	existingCollections := make(map[string]bool)
	for _, col := range collections {
		existingCollections[col.Name] = true
	}

	return existingCollections, nil
}

// createCollectionIfNotExists ensures that a collection exists before trying to use it.
func (b *BaseAPI[indexDocument, returnType]) createCollectionIfNotExists(ctx context.Context, schema *api.CollectionSchema, collectionName string) error {
	// Check if collection already exists
	existingCollections, err := b.fetchExistingCollections(ctx)
	if err != nil {
		return err
	}

	if existingCollections[collectionName] {
		b.l.Info("collection already exists, skipping creation", zap.String("collection", collectionName))
		return nil
	}

	// Set the collection name and create it
	schema.Name = collectionName
	_, err = b.client.Collections().Create(ctx, schema)
	if err != nil {
		b.l.Error("failed to create collection", zap.String("collection", collectionName), zap.Error(err))
		return err
	}

	b.l.Info("created new collection", zap.String("collection", collectionName))
	return nil
}
