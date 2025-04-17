package typesenseapi

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	pkgx "github.com/foomo/typesense/pkg"
	"github.com/typesense/typesense-go/v3/typesense"
	"github.com/typesense/typesense-go/v3/typesense/api"
	"github.com/typesense/typesense-go/v3/typesense/api/pointer"
	"go.uber.org/zap"
)

const defaultSearchPresetName = "default"

type DocumentConverter[indexDocument any, returnType any] func(indexDocument) returnType

type BaseAPI[indexDocument any, returnType any] struct {
	l                 *zap.Logger
	client            *typesense.Client
	collections       map[pkgx.IndexID]*api.CollectionSchema
	preset            *api.PresetUpsertSchema
	revisionID        pkgx.RevisionID
	documentConverter DocumentConverter[indexDocument, returnType]
}

func NewBaseAPI[indexDocument any, returnType any](
	l *zap.Logger,
	client *typesense.Client,
	collections map[pkgx.IndexID]*api.CollectionSchema,
	preset *api.PresetUpsertSchema,
	documentConverter DocumentConverter[indexDocument, returnType],
) *BaseAPI[indexDocument, returnType] {
	return &BaseAPI[indexDocument, returnType]{
		l:                 l,
		client:            client,
		collections:       collections,
		preset:            preset,
		documentConverter: documentConverter,
	}
}

// Healthz will check if the revisionID is set
func (b *BaseAPI[indexDocument, returnType]) Healthz(_ context.Context) error {
	if b.revisionID == "" {
		return errors.New("revisionID not set")
	}
	return nil
}

// Indices returns a list of all configured index IDs
func (b *BaseAPI[indexDocument, returnType]) Indices() ([]pkgx.IndexID, error) {
	if len(b.collections) == 0 {
		return nil, errors.New("no collections configured")
	}
	indices := make([]pkgx.IndexID, 0, len(b.collections))
	for index := range b.collections {
		indices = append(indices, index)
	}
	return indices, nil
}

// Initialize
// This function ensures that a new collection is created for each alias on every run
// and updates aliases to point to the latest revision.
//
// example:
//
//			b.collections := map[IndexID]*api.CollectionSchema{
//				"www-bks-at-de": {
//					Name: "www-bks-at-de",
//			  },
//				"digital-bks-at-de": {
//					Name: "digital-bks-at-de",
//			  }
//		  }
//
//	   There should be 2 aliases: "www-bks-at-de" and "digital-bks-at-de".
//	   There should be at least 2 collections, one for each alias.
//	   The collection names are concatenated with the revision ID: "www-bks-at-de-2021-01-01-12".
//	   The revision ID is a timestamp in the format "YYYY-MM-DD-HH". If multiple collections are available,
//	   the latest revision ID can be identified by the latest timestamp value.
//
// Additionally, ensure that the configured search preset is present.
// The system is considered valid if there is one alias for each collection and the collections
// are correctly linked to their respective aliases.
// The function sets the revisionID that is currently linked to the aliases internally.
func (b *BaseAPI[indexDocument, returnType]) Initialize(ctx context.Context) (pkgx.RevisionID, error) {
	b.l.Info("initializing typesense collections and aliases...")

	// Step 1: Check Typesense connection
	if _, err := b.client.Health(ctx, 5*time.Second); err != nil {
		b.l.Error("typesense health check failed", zap.Error(err))
		return "", err
	}

	// Step 2: Retrieve existing aliases and collections
	aliases, err := b.client.Aliases().Retrieve(ctx)
	if err != nil {
		b.l.Error("failed to retrieve aliases", zap.Error(err))
		return "", err
	}

	existingCollections, err := b.fetchExistingCollections(ctx)
	if err != nil {
		return "", err
	}

	// Step 3: Track latest revisions per alias
	latestRevisions := make(map[pkgx.IndexID]pkgx.RevisionID)
	aliasMappings := make(map[pkgx.IndexID]string) // Tracks alias-to-collection mappings

	for _, alias := range aliases {
		collectionName := alias.CollectionName
		indexID := pkgx.IndexID(*alias.Name)
		revisionID := extractRevisionID(collectionName, string(indexID))

		// Ensure alias points to an existing collection
		if revisionID != "" && existingCollections[collectionName] {
			latestRevisions[indexID] = revisionID
			aliasMappings[indexID] = collectionName
		} else {
			b.l.Warn("alias points to missing collection, resetting", zap.String("alias", string(indexID)))
		}
	}

	// Step 4: Ensure all aliases are correctly mapped to collections and create a new revision
	newRevisionID := b.generateRevisionID()
	b.l.Info("generated new revision", zap.String("revisionID", string(newRevisionID)))

	for indexID, schema := range b.collections {
		collectionName := formatCollectionName(indexID, newRevisionID)

		b.l.Warn("creating new collection & alias",
			zap.String("index", string(indexID)),
			zap.String("new_collection", collectionName),
		)

		// Create new collection
		if err := b.createCollectionIfNotExists(ctx, schema, collectionName); err != nil {
			return "", err
		}

		// Update alias to point to new collection
		if err := b.ensureAliasMapping(ctx, indexID, collectionName); err != nil {
			return "", err
		}
	}

	// Step 5: Set the latest revision ID and return
	b.revisionID = newRevisionID

	// Step 6: Ensure search preset is present
	if b.preset != nil {
		_, err := b.client.Presets().Upsert(ctx, defaultSearchPresetName, b.preset)
		if err != nil {
			b.l.Error("failed to upsert search preset", zap.Error(err))
			return "", err
		}
	}

	b.l.Info("initialization completed", zap.String("revisionID", string(b.revisionID)))

	return b.revisionID, nil
}

func (b *BaseAPI[indexDocument, returnType]) UpsertDocuments(
	ctx context.Context,
	revisionID pkgx.RevisionID,
	indexID pkgx.IndexID,
	documents []*indexDocument,
) error {
	if len(documents) == 0 {
		b.l.Warn("no documents provided for upsert", zap.String("index", string(indexID)))
		return nil
	}

	collectionName := formatCollectionName(indexID, revisionID)

	// Convert []indexDocument to []interface{} to satisfy Import() method
	docInterfaces := make([]interface{}, len(documents))
	for i, doc := range documents {
		b.l.Info("doc", zap.Any("doc", doc))
		docInterfaces[i] = doc
	}

	// Perform bulk upsert using Import()
	params := &api.ImportDocumentsParams{
		Action: (*api.IndexAction)(pointer.String("upsert")),
	}

	importResults, err := b.client.Collection(collectionName).Documents().Import(ctx, docInterfaces, params)
	if err != nil {
		b.l.Error("failed to bulk upsert documents", zap.String("collection", collectionName), zap.Error(err))
		return err
	}

	// Log success and failure counts
	successCount, failureCount := 0, 0
	for _, result := range importResults {
		if result.Success {
			successCount++
		} else {
			failureCount++
			b.l.Warn("document failed to upsert",
				zap.String("collection", collectionName),
				zap.String("error", result.Error),
			)
		}
	}

	b.l.Info("bulk upsert completed",
		zap.String("collection", collectionName),
		zap.Int("successful_documents", successCount),
		zap.Int("failed_documents", failureCount),
	)
	return nil
}

// CommitRevision this is called when all the documents have been upserted
// it will update the aliases to point to the new revision
// additionally it will remove all old collections that are not linked to an alias
// keeping only the latest revision and the one before
func (b *BaseAPI[indexDocument, returnType]) CommitRevision(ctx context.Context, revisionID pkgx.RevisionID) error {
	for indexID := range b.collections {
		alias := string(indexID)
		newCollectionName := formatCollectionName(indexID, revisionID)

		// Step 1: Update the alias to point to the new collection
		_, err := b.client.Aliases().Upsert(ctx, alias,
			&api.CollectionAliasSchema{
				CollectionName: newCollectionName,
			})
		if err != nil {
			b.l.Error("failed to update alias", zap.String("alias", alias), zap.Error(err))
			return err
		}
		b.l.Info("updated alias", zap.String("alias", alias), zap.String("collection", newCollectionName))

		// Step 2: Clean up old collections (keep only the last two)
		err = b.pruneOldCollections(ctx, alias, newCollectionName)
		if err != nil {
			b.l.Error("failed to clean up old collections", zap.String("alias", alias), zap.Error(err))
		}
	}

	return nil
}

// RevertRevision will remove the collections created for the given revisionID
func (b *BaseAPI[indexDocument, returnType]) RevertRevision(ctx context.Context, revisionID pkgx.RevisionID) error {
	for indexID := range b.collections {
		collectionName := formatCollectionName(indexID, revisionID)

		// Step 1: Delete the collection safely
		_, err := b.client.Collection(collectionName).Delete(ctx)
		if err != nil {
			b.l.Error("failed to delete collection", zap.String("collection", collectionName), zap.Error(err))
			return err
		}

		b.l.Info("reverted and deleted collection", zap.String("collection", collectionName))
	}

	return nil
}

// SimpleSearch will perform a search operation on the given index
// it will return the documents and the scores
func (b *BaseAPI[indexDocument, returnType]) SimpleSearch(
	ctx context.Context,
	index pkgx.IndexID,
	q string,
	filterBy map[string][]string,
	page, perPage int,
	sortBy string,
) ([]returnType, pkgx.Scores, int, error) {
	// Call buildSearchParams but also set QueryBy explicitly
	parameters := buildSearchParams(q, filterBy, page, perPage, sortBy)
	parameters.QueryBy = pointer.String("title")

	return b.ExpertSearch(ctx, index, parameters)
}

// ExpertSearch performs a search operation on the given index
// It returns the converted documents, scores, and totalResults
func (b *BaseAPI[indexDocument, returnType]) ExpertSearch(
	ctx context.Context,
	indexID pkgx.IndexID,
	parameters *api.SearchCollectionParams,
) ([]returnType, pkgx.Scores, int, error) {
	if parameters == nil {
		b.l.Error("search parameters are nil")
		return nil, nil, 0, errors.New("search parameters cannot be nil")
	}

	collectionName := string(indexID) // digital-bks-at-de
	searchResponse, err := b.client.Collection(collectionName).Documents().Search(ctx, parameters)
	if err != nil {
		b.l.Error("failed to perform search", zap.String("index", collectionName), zap.Error(err))
		return nil, nil, 0, err
	}

	// Extract totalResults from the search response
	totalResults := *searchResponse.Found

	// Ensure Hits is not empty before proceeding
	if searchResponse.Hits == nil || len(*searchResponse.Hits) == 0 {
		b.l.Warn("search response contains no hits", zap.String("index", collectionName))
		return nil, nil, totalResults, nil
	}

	results := make([]returnType, len(*searchResponse.Hits))
	scores := make(pkgx.Scores)

	for i, hit := range *searchResponse.Hits {
		if hit.Document == nil {
			b.l.Warn("hit document is nil", zap.String("index", collectionName))
			continue
		}

		docMap := *hit.Document

		// Extract document ID safely
		docID, ok := docMap["id"].(string)
		if !ok {
			b.l.Warn("missing or invalid document ID in search result")
			continue
		}

		// Convert raw document (map) to indexDocument struct
		hitJSON, err := json.Marshal(docMap)
		if err != nil {
			b.l.Warn("failed to marshal document to JSON", zap.String("index", collectionName), zap.Error(err))
			continue
		}

		var rawDoc indexDocument
		if err := json.Unmarshal(hitJSON, &rawDoc); err != nil {
			b.l.Warn("failed to unmarshal JSON into indexDocument", zap.String("index", collectionName), zap.Error(err))
			continue
		}

		// Convert the raw document using documentConverter
		convertedDoc := b.documentConverter(rawDoc)
		results[i] = convertedDoc

		// Extract search score
		index := 0
		if hit.TextMatchInfo != nil && hit.TextMatchInfo.Score != nil {
			if score, err := strconv.Atoi(*hit.TextMatchInfo.Score); err == nil {
				index = score
			} else {
				b.l.Warn("invalid score value", zap.String("score", *hit.TextMatchInfo.Score), zap.Error(err))
			}
		}

		scores[pkgx.DocumentID(docID)] = pkgx.Score{
			ID:    pkgx.DocumentID(docID),
			Index: index,
		}
	}

	b.l.Info("search completed",
		zap.String("index", collectionName),
		zap.Int("results_count", len(results)),
		zap.Int("total_results", totalResults),
	)

	return results, scores, totalResults, nil
}
