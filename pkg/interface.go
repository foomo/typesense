package typesense

import (
	"context"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

type API[indexDocument any, returnType any] interface {
	// this will prepare new indices with the given schema and the index IDs configured for the API
	NewRevision() (RevisionID, error)
	CommitRevision(revisionID RevisionID) error
	RevertRevision(revisionID RevisionID) error
	UpsertDocuments(revisionID RevisionID, indexID IndexID, documents []indexDocument) error

	// this will check the typesense connection and initialize the indices
	// should be run directly in a main.go or similar to ensure the connection is working
	Initialize() (RevisionID, error)

	// perform a search operation on the given index
	SimpleSearch(
		index IndexID,
		q string,
		filterBy map[string]string,
		page, perPage int,
		sortBy string,
	) ([]returnType, Scores, error)
	ExpertSearch(index IndexID, parameters *api.SearchCollectionParams) ([]returnType, Scores, error)
	Healthz(ctx context.Context) error
	Indices() ([]IndexID, error)
}

type IndexerInterface[indexDocument any, returnType any] interface {
	Run(ctx context.Context) error
}

type DocumentProvider[indexDocument any] interface {
	Provide(ctx context.Context, index IndexID) ([]indexDocument, error)
	ProvidePaged(ctx context.Context, index IndexID, offset int) ([]indexDocument, int, error)
}
