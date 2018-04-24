package api

import (
	"context"

	"github.com/ONSdigital/dp-dataset-api/store"
	"github.com/ONSdigital/dp-search-api/searchoutputqueue"
	clientsidentity "github.com/ONSdigital/go-ns/clients/identity"
	"github.com/ONSdigital/go-ns/healthcheck"
	"github.com/ONSdigital/go-ns/identity"
	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/go-ns/server"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
)

var httpServer *server.Server

// API provides an interface for the routes
type API interface {
	CreateSearchAPI(string, *mux.Router, store.DataStore) *SearchAPI
}

// DownloadsGenerator pre generates full file downloads for the specified dataset/edition/version
type DownloadsGenerator interface {
	Generate(datasetID, instanceID, edition, version string) error
}

// OutputQueue - An interface used to queue search outputs
type OutputQueue interface {
	Queue(output *searchoutputqueue.Search) error
}

// SearchAPI manages searches across indices
type SearchAPI struct {
	datasetAPIClient       DatasetAPIer
	datasetAPIClientNoAuth DatasetAPIer
	defaultMaxResults      int
	elasticsearch          Elasticsearcher
	hasPrivateEndpoints    bool
	host                   string
	internalToken          string
	router                 *mux.Router
	searchOutputQueue      OutputQueue
}

// CreateSearchAPI manages all the routes configured to API
func CreateSearchAPI(
	host, bindAddr, authAPIURL string, errorChan chan error, searchOutputQueue OutputQueue,
	datasetAPIClient, datasetAPIClientNoAuth DatasetAPIer,
	elasticsearch Elasticsearcher, defaultMaxResults int, hasPrivateEndpoints bool, serviceAuthToken string,
) {
	router := mux.NewRouter()
	routes(host, authAPIURL, router, searchOutputQueue, datasetAPIClient, datasetAPIClientNoAuth, elasticsearch, defaultMaxResults, hasPrivateEndpoints)

	authClient := clientsidentity.NewAPIClient(nil, authAPIURL)

	identityHandler := identity.HandlerForHTTPClient(true, authClient)
	alice := alice.New(identityHandler).Then(router)

	httpServer = server.New(bindAddr, alice)
	// Disable this here to allow service to manage graceful shutdown of the entire app.
	httpServer.HandleOSSignals = false

	go func() {
		log.Debug("Starting api...", nil)
		if err := httpServer.ListenAndServe(); err != nil {
			log.ErrorC("api http server returned error", err, nil)
			errorChan <- err
		}
	}()
}

func routes(host, authAPIURL string, router *mux.Router, searchOutputQueue OutputQueue, datasetAPIClient, datasetAPIClientNoAuth DatasetAPIer, elasticsearch Elasticsearcher, defaultMaxResults int, hasPrivateEndpoints bool) *SearchAPI {

	api := SearchAPI{
		datasetAPIClient:       datasetAPIClient,
		datasetAPIClientNoAuth: datasetAPIClientNoAuth,
		defaultMaxResults:      defaultMaxResults,
		elasticsearch:          elasticsearch,
		hasPrivateEndpoints:    hasPrivateEndpoints,
		searchOutputQueue:      searchOutputQueue,
		host:                   host,
		router:                 router,
	}

	router.Path("/healthcheck").Methods("GET").HandlerFunc(healthcheck.Do)

	api.router.HandleFunc("/search/datasets/{id}/editions/{edition}/versions/{version}/dimensions/{name}", api.getSearch).Methods("GET")

	if hasPrivateEndpoints {
		api.router.HandleFunc("/search/instances/{instance_id}/dimensions/{dimension}", identity.Check(api.createSearchIndex)).Methods("PUT")
		api.router.HandleFunc("/search/instances/{instance_id}/dimensions/{dimension}", identity.Check(api.deleteSearchIndex)).Methods("DELETE")
	}

	return &api
}

// Close represents the graceful shutting down of the http server
func Close(ctx context.Context) error {
	if err := httpServer.Shutdown(ctx); err != nil {
		return err
	}
	log.Info("graceful shutdown of http server complete", nil)
	return nil
}
