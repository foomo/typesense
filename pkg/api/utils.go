package typesenseapi

import (
	"strings"

	"github.com/typesense/typesense-go/v3/typesense/api"
	"github.com/typesense/typesense-go/v3/typesense/api/pointer"
)

// getSearchCollectionParameters will return the search collection parameters
// this is meant as a utility function to create the search collection parameters
// for the typesense search API without any knowledge of the typesense API
func getSearchCollectionParameters(
	q string,
	filterBy map[string]string,
	page, perPage int,
	sortBy string,
) *api.SearchCollectionParams {
	parameters := &api.SearchCollectionParams{}
	parameters.Q = pointer.String(q)
	if filterByString := getFilterByString(filterBy); filterByString != "" {
		parameters.FilterBy = pointer.String(filterByString)
	}
	parameters.Page = pointer.Int(page)
	parameters.PerPage = pointer.Int(perPage)
	if sortBy != "" {
		parameters.SortBy = pointer.String(sortBy)
	}

	return parameters
}

func getFilterByString(filterBy map[string]string) string {
	if filterBy == nil {
		return ""
	}
	filterByString := []string{}
	for key, value := range filterBy {
		filterByString = append(filterByString, key+":="+value)
	}
	return strings.Join(filterByString, "&&")
}
