package typesenseindexing

import (
	"context"
	"fmt"
	"slices"

	"github.com/foomo/contentserver/client"
	"github.com/foomo/contentserver/content"
	typesense "github.com/foomo/typesense/pkg"
	"go.uber.org/zap"
)

type ContentServer[indexDocument any] struct {
	l                     *zap.Logger
	contentserverClient   *client.Client
	documentProviderFuncs map[typesense.DocumentType]typesense.DocumentProviderFunc[indexDocument]
	supportedMimeTypes    []string
}

func NewContentServer[indexDocument any](
	l *zap.Logger,
	client *client.Client,
	documentProviderFuncs map[typesense.DocumentType]typesense.DocumentProviderFunc[indexDocument],
	supportedMimeTypes []string,
) *ContentServer[indexDocument] {
	return &ContentServer[indexDocument]{
		l:                     l,
		contentserverClient:   client,
		documentProviderFuncs: documentProviderFuncs,
		supportedMimeTypes:    supportedMimeTypes,
	}
}

func (c ContentServer[indexDocument]) Provide(
	ctx context.Context,
	indexID typesense.IndexID,
) ([]*indexDocument, error) {
	documentInfos, err := c.getDocumentIDsByIndexID(ctx, indexID)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(documentInfos))
	for _, documentInfo := range documentInfos {
		ids = append(ids, string(documentInfo.DocumentID))
	}

	uriMap, err := c.contentserverClient.GetURIs(ctx, string(indexID), ids)
	if err != nil {
		c.l.Error("failed to get URIs", zap.Error(err))
		return nil, err
	}

	documents := make([]*indexDocument, len(documentInfos))
	for index, documentInfo := range documentInfos {
		if documentProvider, ok := c.documentProviderFuncs[documentInfo.DocumentType]; !ok {
			c.l.Warn("no document provider available for document type", zap.String("documentType", string(documentInfo.DocumentType)))
		} else {
			document, err := documentProvider(ctx, indexID, documentInfo.DocumentID, uriMap)
			if err != nil {
				c.l.Error(
					"index document not created",
					zap.Error(err),
					zap.String("documentID", string(documentInfo.DocumentID)),
					zap.String("documentType", string(documentInfo.DocumentType)),
				)
				continue
			}
			if document != nil {
				documents[index] = document
			}
		}
	}
	return documents, nil
}

func (c ContentServer[indexDocument]) ProvidePaged(
	ctx context.Context,
	indexID typesense.IndexID,
	offset int,
) ([]*indexDocument, int, error) {
	panic("implement me")
}

func (c ContentServer[indexDocument]) getDocumentIDsByIndexID(
	ctx context.Context,
	indexID typesense.IndexID,
) ([]typesense.DocumentInfo, error) {
	// get the contentserver dimension defined by indexID
	// create the list of document infos
	repo, err := c.contentserverClient.GetRepo(ctx)
	if err != nil {
		return nil, err
	}
	rootRepoNode, ok := repo[string(indexID)]
	if !ok {
		return nil, fmt.Errorf("contenserver dimension %s not found", indexID)
	}

	nodeMap := createFlatRepoNodeMap(rootRepoNode, map[string]*content.RepoNode{})
	documentInfos := make([]typesense.DocumentInfo, 0, len(nodeMap))
	for _, repoNode := range nodeMap {
		if slices.Contains(c.supportedMimeTypes, repoNode.MimeType) {
			documentInfos = append(documentInfos, typesense.DocumentInfo{
				DocumentType: typesense.DocumentType(repoNode.MimeType),
				DocumentID:   typesense.DocumentID(repoNode.ID),
			})
		}
	}

	return documentInfos, nil
}

// createFlatRepoNodeMap recursively retrieves all nodes from the tree and returns them in a flat map.
func createFlatRepoNodeMap(node *content.RepoNode, nodeMap map[string]*content.RepoNode) map[string]*content.RepoNode {
	if node == nil {
		return nodeMap
	}
	// Add the current node to the list.
	nodeMap[node.ID] = node
	// Recursively process child nodes.
	for _, child := range node.Nodes {
		nodeMap = createFlatRepoNodeMap(child, nodeMap)
	}
	return nodeMap
}
