package typesenseindexing

import (
	"context"
	"fmt"

	"slices"

	contentserverclient "github.com/foomo/contentserver/client"
	"github.com/foomo/contentserver/content"
	pkgx "github.com/foomo/typesense/pkg"
	"go.uber.org/zap"
)

type ContentServer[indexDocument any] struct {
	l                     *zap.Logger
	contentserverClient   *contentserverclient.Client
	documentProviderFuncs map[pkgx.DocumentType]pkgx.DocumentProviderFunc[indexDocument]
	supportedMimeTypes    []string
}

func NewContentServer[indexDocument any](
	l *zap.Logger,
	client *contentserverclient.Client,
	documentProviderFuncs map[pkgx.DocumentType]pkgx.DocumentProviderFunc[indexDocument],
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
	indexID pkgx.IndexID,
) ([]*indexDocument, error) {
	documentInfos, err := c.getDocumentIDsByIndexID(ctx, indexID)
	if err != nil {
		return nil, err
	}

	urlsByIDs, err := c.fetchURLsByDocumentIDs(ctx, indexID, documentInfos)
	if err != nil {
		return nil, err
	}

	documents := make([]*indexDocument, len(documentInfos))
	for index, documentInfo := range documentInfos {
		if documentProvider, ok := c.documentProviderFuncs[documentInfo.DocumentType]; !ok {
			c.l.Warn("no document provider available for document type", zap.String("documentType", string(documentInfo.DocumentType)))
		} else {
			document, err := documentProvider(ctx, indexID, documentInfo.DocumentID, urlsByIDs)
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
	indexID pkgx.IndexID,
	offset int,
) ([]*indexDocument, int, error) {
	panic("implement me")
}

func (c ContentServer[indexDocument]) getDocumentIDsByIndexID(
	ctx context.Context,
	indexID pkgx.IndexID,
) ([]pkgx.DocumentInfo, error) {
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
	documentInfos := make([]pkgx.DocumentInfo, 0, len(nodeMap))
	for _, repoNode := range nodeMap {
		if repoNode.Hidden || !slices.Contains(c.supportedMimeTypes, repoNode.MimeType) {
			c.l.Warn("Skipping document indexing",
				zap.String("path", repoNode.URI),
				zap.String("mimeType", repoNode.MimeType),
				zap.Bool("hidden", repoNode.Hidden),
			)
			continue
		}

		documentInfos = append(documentInfos, pkgx.DocumentInfo{
			DocumentType: pkgx.DocumentType(repoNode.MimeType),
			DocumentID:   pkgx.DocumentID(repoNode.ID),
		})
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

func (c ContentServer[indexDocument]) fetchURLsByDocumentIDs(
	ctx context.Context,
	indexID pkgx.IndexID,
	documentInfos []pkgx.DocumentInfo,
) (map[pkgx.DocumentID]string, error) {
	ids := make([]string, len(documentInfos))

	for i, documentInfo := range documentInfos {
		ids[i] = string(documentInfo.DocumentID)
	}

	uriMap, err := c.contentserverClient.GetURIs(ctx, string(indexID), ids)
	if err != nil {
		c.l.Error("failed to get URIs", zap.Error(err))
		return nil, err
	}

	return convertMapStringToDocumentID(uriMap), nil
}

func convertMapStringToDocumentID(input map[string]string) map[pkgx.DocumentID]string {
	output := make(map[pkgx.DocumentID]string, len(input))
	for key, value := range input {
		output[pkgx.DocumentID(key)] = value
	}
	return output
}
