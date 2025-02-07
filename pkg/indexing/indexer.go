package typesenseindexing

import (
	"context"
	typesense "github.com/foomo/typesense/pkg"
	"go.uber.org/zap"
)

type BaseIndexer[indexDocument any, returnType any] struct {
	l                *zap.Logger
	typesenseAPI     typesense.API[indexDocument, returnType]
	documentProvider typesense.DocumentProvider[indexDocument]
}

func NewBaseIndexer[indexDocument any, returnType any](
	l *zap.Logger,
	typesenseAPI typesense.API[indexDocument, returnType],
	documentProvider typesense.DocumentProvider[indexDocument],
) *BaseIndexer[indexDocument, returnType] {
	return &BaseIndexer[indexDocument, returnType]{
		l:                l,
		typesenseAPI:     typesenseAPI,
		documentProvider: documentProvider,
	}
}

func (b *BaseIndexer[indexDocument, returnType]) Healthz(ctx context.Context) error {
	return b.typesenseAPI.Healthz(ctx)
}

func (b *BaseIndexer[indexDocument, returnType]) Run(ctx context.Context) error {
	// return error if the health check fails
	if err := b.Healthz(ctx); err != nil {
		return err
	}

	// create a new revision
	revisionID, err := b.typesenseAPI.NewRevision()
	if err != nil {
		return err
	}

	// get the configured indices from the typesense API
	indices, err := b.typesenseAPI.Indices()
	if err != nil {
		return err
	}

	// set a variable to check if the upserting of documents was successful
	tainted := false

	// for each index, get the documents from the document provider and upsert them
	for _, indexID := range indices {
		documents, err := b.documentProvider.Provide(ctx, indexID)
		if err != nil {
			return err
		}

		err = b.typesenseAPI.UpsertDocuments(revisionID, indexID, documents)
		if err != nil {
			b.l.Error(
				"failed to upsert documents",
				zap.Error(err),
				zap.String("index", string(indexID)),
				zap.String("revision", string(revisionID)),
				zap.Int("documents", len(documents)),
			)
			tainted = true
			break
		}
	}

	if !tainted {
		// commit the revision if no errors occurred
		err = b.typesenseAPI.CommitRevision(revisionID)
		if err != nil {
			return err
		}
	} else {
		// revert the revision if errors occurred
		err = b.typesenseAPI.RevertRevision(revisionID)
		if err != nil {
			return err
		}
	}

	return nil
}
