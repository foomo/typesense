package typesense

import (
	"context"

	"github.com/typesense/typesense-go/v3/typesense/api"
)

type RevisionID string
type Query string
type IndexID string
type DocumentID string
type DocumentType string

type Scores map[DocumentID]Score

type Score struct {
	ID    DocumentID
	Index int
}

type DocumentProviderFunc[indexDocument any] func(
	ctx context.Context,
	indexID IndexID,
	documentID DocumentID,
	urlsByIDs map[DocumentID]string,
) (*indexDocument, error)

type DocumentInfo struct {
	DocumentType DocumentType
	DocumentID   DocumentID
}

type SearchParameters struct {
	Query      string
	Page       int
	PresetName string
	Modify     func(params *api.SearchCollectionParams)
}
