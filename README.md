# Typesense API
[![Build Status](https://github.com/foomo/typesense/actions/workflows/test.yml/badge.svg?branch=main&event=push)](https://github.com/foomo/typesense/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/foomo/typesense)](https://goreportcard.com/report/github.com/foomo/typesense)
[![GoDoc](https://godoc.org/github.com/foomo/typesense?status.svg)](https://godoc.org/github.com/foomo/typesense)

## Overview
This package provides an API for managing and searching Typesense collections. It offers functionalities for initializing, indexing, searching, and maintaining Typesense collections and aliases.

## Features
- **Initialization**: Ensures that all aliases point to the latest revision-based collections.
- **Health Check**: Verifies if the Typesense client is operational.
- **Index Management**: Lists, creates, and updates index collections.
- **Document Upsertion**: Bulk upsert support for indexing documents.
- **Search Operations**: Provides simple and advanced search capabilities.
- **Revision Management**: Supports committing and reverting indexing revisions.

## Installation
To use this package, add it as a dependency in your Go project:
```sh
 go get github.com/foomo/typesense
```

## Usage example

```go
import (
	"context"
	"time"

	"github.com/foomo/contentserver/client"
	"github.com/foomo/keel/config"
	"github.com/foomo/keel/log"
	typesenseapi "github.com/foomo/typesense/pkg/api"
	typesenseindexing "github.com/foomo/typesense/pkg/indexing"

	"github.com/typesense/typesense-go/v3/typesense"
)

func main() {
	ctx := context.Background()
	l := log.Logger()

	// contentserver client
	csClient, errClient := client.NewHTTPClient("contentserver_url")
	log.Must(l, errClient, "could not get contentserver client")

	// contentful client
	// provide list of content types
	cfClients := contentful.NewDefaultContentfulClients(ctx, l, contentful_types, true)
	cfClients.UpdateCache()
	cfClients.Client.ClientStats()

	// create typesense client
	typesenseClient := typesense.NewClient(
		typesense.WithConnectionTimeout(2*time.Minute),
		typesense.WithServer("typesense_server"),
		typesense.WithAPIKey("typesense_api_key"),
	)

	// configure document provider
	documentProvider := typesenseindexing.NewContentServer(
		l, csClient,
		GetDocumentProviderFunctions(...),  // retrieve document provider functions
		supportedMimeTypes,                 // provide supported mime types
	)

	// create typesense api
	// create indexDocument and returnType
	api := typesenseapi.NewBaseAPI[indexDocument, returnType](
		l,
		typesenseClient,
		collectionSchemas,  //map[IndexID]*api.CollectionSchema
		presetUpsertSchema, //*api.PresetUpsertSchema
	)

	// create typesense indexer
	indexer := typesenseindexing.NewBaseIndexer(
		l,
		api,
		documentProvider,
	)

	// run indexer
	err := indexer.Run(ctx)
	log.Must(l, err, "could not run indexer")
}
```

### Health Check
```go
err := apiInstance.Healthz(context.Background())
if err != nil {
	log.Fatalf("Health check failed: %v", err)
}
```

### Searching Documents
#### Simple Search
```go
results, scores, total, err := apiInstance.SimpleSearch(context.Background(), "products", "laptop", nil, 1, 10, "price:desc")
if err != nil {
	log.Fatalf("Search failed: %v", err)
}
log.Printf("Found %d results", total)
```

#### Advanced Search
```go
searchParams := &api.SearchCollectionParams{
	Q:       pointer.String("laptop"),
	SortBy:  pointer.String("price:desc"),
}

results, scores, total, err = apiInstance.ExpertSearch(context.Background(), "products", searchParams)
if err != nil {
	log.Fatalf("Advanced search failed: %v", err)
}
log.Printf("Found %d results", total)
```

## How to Contribute

Please refer to the [CONTRIBUTING](.github/CONTRIBUTING.md) details and follow the [CODE_OF_CONDUCT](.github/CODE_OF_CONDUCT.md) and [SECURITY](.github/SECURITY.md) guidelines.

## License

Distributed under MIT License, please see license file within the code for more details.

_Made with ♥ [foomo](https://www.foomo.org) by [bestbytes](https://www.bestbytes.com)_
