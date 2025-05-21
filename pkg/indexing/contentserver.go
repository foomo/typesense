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

const ContentserverDataAttributeNoIndex = "typesenseIndexing-noIndex"

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

// Provide retrieves documents for the given indexID from the content server.
// It fetches the document IDs, retrieves the URLs for those IDs, and then uses the
// document provider functions to create the documents.
// The documents are returned as a slice of pointers to the indexDocument type.
// If a document provider function is not available for a specific document type,
// a warning is logged and that document is skipped.
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
			c.l.Warn(
				"no document provider available for document type",
				zap.String("documentType", string(documentInfo.DocumentType)),
			)
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

// ProvidePaged
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
		if !includeNode(c.supportedMimeTypes, repoNode) {
			c.l.Debug("skipping document indexing",
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

// fetchURLsByDocumentIDs fetches the URLs for the given document IDs from the content server.
// It uses the contentserverClient to retrieve the URIs and maps them to DocumentID.
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

// convertMapStringToDocumentID converts a map with string keys to a map with DocumentID keys.
// The keys in the input map are converted to DocumentID type, while the values remain strings.
func convertMapStringToDocumentID(input map[string]string) map[pkgx.DocumentID]string {
	output := make(map[pkgx.DocumentID]string, len(input))
	for key, value := range input {
		output[pkgx.DocumentID(key)] = value
	}
	return output
}

// includeNode checks if the node should be included in the indexing process.
// It checks if the node is nil, if it has the noIndex attribute set to true,
// and if its mime type is in the list of supported mime types.
func includeNode(supportedMimeTypes []string, node *content.RepoNode) bool {
	if node == nil {
		return false
	}
	if noIndex, noIndexSet := node.Data[ContentserverDataAttributeNoIndex].(bool); noIndexSet && noIndex {
		return false
	}
	if !slices.Contains(supportedMimeTypes, node.MimeType) {
		return false
	}
	return true
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
