package typesenseapi

import (
	"context"
	"errors"
	typesense2 "github.com/foomo/typesense/pkg"
	"github.com/typesense/typesense-go/v3/typesense"
	"github.com/typesense/typesense-go/v3/typesense/api"
	"go.uber.org/zap"
)

const defaultSearchPresetName = "default"

type BaseAPI[indexDocument any, returnType any] struct {
	l           *zap.Logger
	client      *typesense.Client
	collections map[typesense2.IndexID]*api.CollectionSchema
	preset      *api.PresetUpsertSchema

	revisionID typesense2.RevisionID
}

func NewBaseAPI[indexDocument any, returnType any](
	l *zap.Logger,
	client *typesense.Client,
	collections map[typesense2.IndexID]*api.CollectionSchema,
	preset *api.PresetUpsertSchema,
) *BaseAPI[indexDocument, returnType] {
	return &BaseAPI[indexDocument, returnType]{
		l:           l,
		client:      client,
		collections: collections,
		preset:      preset,
	}
}

// Healthz will check if the revisionID is set
func (b *BaseAPI[indexDocument, returnType]) Healthz(_ context.Context) error {
	if b.revisionID == "" {
		return errors.New("revisionID not set")
	}
	return nil
}

// Healthz will check if the revisionID is set
func (b *BaseAPI[indexDocument, returnType]) Indices() ([]typesense2.IndexID, error) {
	if len(b.collections) == 0 {
		return nil, errors.New("no collections configured")
	}
	indices := make([]typesense2.IndexID, 0, len(b.collections))
	for index := range b.collections {
		indices = append(indices, index)
	}
	return indices, nil
}

// Initialize
// will check the typesense connection and state of the colllections and aliases
// if the collections and aliases are not in the correct state it will create new collections and aliases
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
//	   there should be 2 aliases "www-bks-at-de" and "digital-bks-at-de"
//	   there should be at least 2 collections one for each alias
//	   the collection names are concatenated with the revisionID: "www-bks-at-de-2021-01-01-12"
//	   the revisionID is a timestamp in the format "YYYY-MM-DD-HH". If multiple collections are available
//	   the latest revisionID can be identified by the latest timestamp value
//
// Additionally, make sure that the configured search preset is present
// The system is ok if there is one alias for each collection and the collections are linked to the correct alias
// The function will set the revisionID that is currently linked to the aliases internally
func (b *BaseAPI[indexDocument, returnType]) Initialize() (typesense2.RevisionID, error) {
	var revisionID typesense2.RevisionID
	// use b.client.Health() to check the connection

	b.revisionID = revisionID
	return "", nil
}

func (b *BaseAPI[indexDocument, returnType]) NewRevision() (typesense2.RevisionID, error) {
	var revision typesense2.RevisionID

	// create a revisionID based on the current time "YYYY-MM-DD-HH"

	// for all b.collections
	// create a new collection in typesense - IndexID + - + revisionID
	return revision, nil
}

func (b *BaseAPI[indexDocument, returnType]) UpsertDocuments(
	revisionID typesense2.RevisionID,
	indexID typesense2.IndexID,
	documents []indexDocument,
) error {
	// use api to upsert documents
	return nil
}

// CommitRevision this is called when all the documents have been upserted
// it will update the aliases to point to the new revision
// additionally it will remove all old collections that are not linked to an alias
// keeping only the latest revision and the one before
func (b *BaseAPI[indexDocument, returnType]) CommitRevision(revisionID typesense2.RevisionID) error {
	return nil
}

// RevertRevision will remove the collections created for the given revisionID
func (b *BaseAPI[indexDocument, returnType]) RevertRevision(revisionID typesense2.RevisionID) error {
	return nil
}

// SimpleSearch will perform a search operation on the given index
// it will return the documents and the scores
func (b *BaseAPI[indexDocument, returnType]) SimpleSearch(
	index typesense2.IndexID,
	q string,
	filterBy map[string]string,
	page, perPage int,
	sortBy string,
) ([]returnType, typesense2.Scores, error) {
	return b.ExpertSearch(index, getSearchCollectionParameters(q, filterBy, page, perPage, sortBy))
}

// ExpertSearch will perform a search operation on the given index
// it will return the documents and the scores
func (b *BaseAPI[indexDocument, returnType]) ExpertSearch(index typesense2.IndexID, parameters *api.SearchCollectionParams) ([]returnType, typesense2.Scores, error) {
	return nil, nil, nil
}
