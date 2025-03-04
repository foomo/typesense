package typesense

import "context"

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
	uriMap map[string]string,
) (*indexDocument, error)

type DocumentInfo struct {
	DocumentType DocumentType
	DocumentID   DocumentID
}
