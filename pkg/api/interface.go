package typesenseapi

import "github.com/typesense/typesense-go/v3/typesense/api"

type API[indexDocument any, returnDocument any] interface {
	// this will prepare new indices with the given schema and the index IDs configured for the API
	NewRevision() (RevisionID, error)
	CommitRevision(revisionID RevisionID) error
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
	) ([]returnDocument, Scores, error)
	ExpertSearch(index IndexID, parameters *api.SearchCollectionParams) ([]returnDocument, Scores, error)
}
