package typesenseapi

import (
	"context"
	"errors"

	"github.com/typesense/typesense-go/v3/typesense"
	"go.uber.org/zap"

	"github.com/typesense/typesense-go/v3/typesense/api"
)

const defaultSearchPresetName = "default"

type BaseAPI[indexDocument any, returnDocument any] struct {
	l           *zap.Logger
	client      *typesense.Client
	collections map[IndexID]*api.CollectionSchema
	preset      *api.PresetUpsertSchema

	revisionID RevisionID
}

func NewBaseAPI[indexDocument any, returnDocument any](
	l *zap.Logger,
	client *typesense.Client,
	collections map[IndexID]*api.CollectionSchema,
	preset *api.PresetUpsertSchema,
) *BaseAPI[indexDocument, returnDocument] {
	return &BaseAPI[indexDocument, returnDocument]{
		l:           l,
		client:      client,
		collections: collections,
		preset:      preset,
	}
}

// Healthz will check if the revisionID is set
func (b *BaseAPI[indexDocument, returnDocument]) Healthz(_ context.Context) error {
	if b.revisionID == "" {
		return errors.New("revisionID not set")
	}
	return nil
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
func (b *BaseAPI[indexDocument, returnDocument]) Initialize() error {
	var revisionID RevisionID
	// use b.client.Health() to check the connection

	b.revisionID = revisionID
	return nil
}

func (b *BaseAPI[indexDocument, returnDocument]) NewRevision() (RevisionID, error) {
	var revision RevisionID

	// create a revisionID based on the current time "YYYY-MM-DD-HH"

	// for all b.collections
	// create a new collection in typesense - IndexID + - + revisionID
	return revision, nil
}

func (b *BaseAPI[indexDocument, returnDocument]) UpsertDocuments(
	revisionID RevisionID,
	indexID IndexID,
	documents []indexDocument,
) error {
	// use api to upsert documents
	return nil
}

// CommitRevision this is called when all the documents have been upserted
// it will update the aliases to point to the new revision
// additionally it will remove all old collections that are not linked to an alias
// keeping only the latest revision and the one before
func (b *BaseAPI[indexDocument, returnDocument]) CommitRevision(revisionID RevisionID) error {
	return nil
}

// SimpleSearch will perform a search operation on the given index
// it will return the documents and the scores
func (b *BaseAPI[indexDocument, returnDocument]) SimpleSearch(
	index IndexID,
	q string,
	filterBy map[string]string,
	page, perPage int,
	sortBy string,
) ([]returnDocument, Scores, error) {
	return b.ExpertSearch(index, getSearchCollectionParameters(q, filterBy, page, perPage, sortBy))
}

// ExpertSearch will perform a search operation on the given index
// it will return the documents and the scores
func (b *BaseAPI[indexDocument, returnDocument]) ExpertSearch(index IndexID, parameters *api.SearchCollectionParams) ([]returnDocument, Scores, error) {
	return nil, nil, nil
}
