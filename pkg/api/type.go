package typesenseapi

type RevisionID string
type Query string
type IndexID string
type DocumentID string

type Scores map[DocumentID]Score

type Score struct {
	ID    DocumentID
	Index int
}
