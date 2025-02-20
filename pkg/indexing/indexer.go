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
	// Step 1: Ensure Typesense is initialized
	revisionID, err := b.typesenseAPI.Initialize(ctx)
	if err != nil || revisionID == "" {
		b.l.Error("Failed to initialize Typesense", zap.Error(err))
		return err
	}

	// Step 2: Retrieve all configured indices
	indices, err := b.typesenseAPI.Indices()
	if err != nil {
		b.l.Error("Failed to retrieve indices from Typesense", zap.Error(err))
		return err
	}

	// Step 3: Track errors while upserting
	tainted := false
	indexedDocuments := 0

	for _, indexID := range indices {
		// Fetch documents from the provider
		documents, err := b.documentProvider.Provide(ctx, indexID)
		if err != nil {
			b.l.Error("Failed to fetch documents", zap.String("index", string(indexID)), zap.Error(err))
			tainted = true
			continue
		}

		err = b.typesenseAPI.UpsertDocuments(ctx, revisionID, indexID, documents)
		if err != nil {
			b.l.Error(
				"Failed to upsert documents",
				zap.String("index", string(indexID)),
				zap.String("revision", string(revisionID)),
				zap.Int("documents", len(documents)),
				zap.Error(err),
			)
			tainted = true
			continue
		}

		indexedDocuments += len(documents)
		b.l.Info("Successfully upserted documents",
			zap.String("index", string(indexID)),
			zap.Int("count", len(documents)),
		)
	}

	// Step 4: Commit or Revert the Revision
	if !tainted && indexedDocuments > 0 {
		// No errors encountered, commit the revision
		err = b.typesenseAPI.CommitRevision(ctx, revisionID)
		if err != nil {
			b.l.Error("Failed to commit revision", zap.String("revision", string(revisionID)), zap.Error(err))
			return err
		}
		b.l.Info("Successfully committed revision", zap.String("revision", string(revisionID)))
	} else {
		// If errors occurred, revert the revision
		b.l.Warn("Errors detected during upsert, reverting revision", zap.String("revision", string(revisionID)))

		err = b.typesenseAPI.RevertRevision(ctx, revisionID)
		if err != nil {
			b.l.Error("Failed to revert revision", zap.String("revision", string(revisionID)), zap.Error(err))
			return err
		}
		b.l.Info("Successfully reverted revision", zap.String("revision", string(revisionID)))
	}

	return nil
}
