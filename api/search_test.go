package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/ONSdigital/dp-search-api/mocks"
	"github.com/ONSdigital/dp-search-api/models"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	host                = "8080"
	secretKey           = "coffee"
	datasetAPISecretKey = "tea"
	defaultMaxResults   = 20
	brokers             = []string{"localhost:9092"}
	topic               = "testing"
)

func TestGetSearchReturnsOK(t *testing.T) {
	t.Parallel()
	Convey("Given the search query satisfies the search index then return a status 200", t, func() {
		r := httptest.NewRequest("GET", "http://localhost:23100/search/datasets/123/editions/2017/versions/1/dimensions/aggregate?q=term", nil)
		w := httptest.NewRecorder()

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusOK)

		// Check response json
		searchResults := getSearchResults(w.Body)

		So(searchResults.Count, ShouldEqual, 2)
		So(len(searchResults.Items), ShouldEqual, 2)
		So(searchResults.Limit, ShouldEqual, 20)
		So(searchResults.Offset, ShouldEqual, 0)
		So(searchResults.Items[0].Code, ShouldEqual, "frs34g5t98hdd")
		So(searchResults.Items[0].DimensionOptionURL, ShouldEqual, "http://localhost:8080/testing/1")
		So(searchResults.Items[0].HasData, ShouldEqual, true)
		So(searchResults.Items[0].Label, ShouldEqual, "something and someone")
		So(searchResults.Items[0].NumberOfChildren, ShouldEqual, 3)
		So(len(searchResults.Items[0].Matches.Code), ShouldEqual, 1)
		So(searchResults.Items[0].Matches.Code[0].Start, ShouldEqual, 1)
		So(searchResults.Items[0].Matches.Code[0].End, ShouldEqual, 13)
		So(len(searchResults.Items[0].Matches.Label), ShouldEqual, 2)
		So(searchResults.Items[0].Matches.Label[0].Start, ShouldEqual, 1)
		So(searchResults.Items[0].Matches.Label[0].End, ShouldEqual, 9)
		So(searchResults.Items[0].Matches.Label[1].Start, ShouldEqual, 13)
		So(searchResults.Items[0].Matches.Label[1].End, ShouldEqual, 19)
		So(searchResults.Items[1].Code, ShouldEqual, "gt534g5t98hs1")
		So(searchResults.Items[1].DimensionOptionURL, ShouldEqual, "http://localhost:8080/testing/2")
		So(searchResults.Items[1].HasData, ShouldEqual, false)
		So(searchResults.Items[1].Label, ShouldEqual, "something else and someone else")
		So(searchResults.Items[1].NumberOfChildren, ShouldEqual, 10)
		So(len(searchResults.Items[1].Matches.Code), ShouldEqual, 0)
		So(len(searchResults.Items[1].Matches.Label), ShouldEqual, 2)
		So(searchResults.Items[1].Matches.Label[0].Start, ShouldEqual, 1)
		So(searchResults.Items[1].Matches.Label[0].End, ShouldEqual, 9)
		So(searchResults.Items[1].Matches.Label[1].Start, ShouldEqual, 19)
		So(searchResults.Items[1].Matches.Label[1].End, ShouldEqual, 25)
	})

	Convey("Given the search query satisfies the search index when limit and offset parameters are set then return a status 200", t, func() {
		r := httptest.NewRequest("GET", "http://localhost:23100/search/datasets/123/editions/2017/versions/1/dimensions/aggregate?q=term&limit=5&offset=20", nil)
		w := httptest.NewRecorder()

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, 40, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusOK)

		// Check response json
		searchResults := getSearchResults(w.Body)

		So(searchResults.Count, ShouldEqual, 2)
		So(len(searchResults.Items), ShouldEqual, 2)
		So(searchResults.Limit, ShouldEqual, 5)
		So(searchResults.Offset, ShouldEqual, 20)
	})
}

func TestGetSearchFailureScenarios(t *testing.T) {
	t.Parallel()
	Convey("Given search API fails to connect to the dataset API return status 500 (internal service error)", t, func() {
		r := httptest.NewRequest("GET", "http://localhost:23100/search/datasets/123/editions/2017/versions/1/dimensions/aggregate?q=term", nil)
		w := httptest.NewRecorder()

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{InternalServerError: true}, &mocks.Elasticsearch{}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
		So(w.Body.String(), ShouldEqual, "Failed to process the request due to an internal error\n")
	})

	Convey("Given the version document was not found via the dataset API return status 404 (not found)", t, func() {
		r := httptest.NewRequest("GET", "http://localhost:23100/search/datasets/123/editions/2017/versions/1/dimensions/aggregate?q=term", nil)
		w := httptest.NewRecorder()

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{VersionNotFound: true}, &mocks.Elasticsearch{}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, "Resource not found\n")
	})

	Convey("Given the limit parameter in request is not a number return status 400 (bad request)", t, func() {
		r := httptest.NewRequest("GET", "http://localhost:23100/search/datasets/123/editions/2017/versions/1/dimensions/aggregate?q=term&limit=four", nil)
		w := httptest.NewRecorder()

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
		So(w.Body.String(), ShouldEqual, "strconv.Atoi: parsing \"four\": invalid syntax\n")
	})

	Convey("Given the offset parameter in request is not a number return status 400 (bad request)", t, func() {
		r := httptest.NewRequest("GET", "http://localhost:23100/search/datasets/123/editions/2017/versions/1/dimensions/aggregate?q=term&offset=fifty", nil)
		w := httptest.NewRecorder()

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
		So(w.Body.String(), ShouldEqual, "strconv.Atoi: parsing \"fifty\": invalid syntax\n")
	})

	Convey("Given the query parameter, q does not exist in request return status 400 (bad request)", t, func() {
		r := httptest.NewRequest("GET", "http://localhost:23100/search/datasets/123/editions/2017/versions/1/dimensions/aggregate", nil)
		w := httptest.NewRecorder()

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
		So(w.Body.String(), ShouldEqual, "search term empty\n")
	})

	Convey("Given the offset parameter exceeds the default maximum results return status 400 (bad request)", t, func() {
		r := httptest.NewRequest("GET", "http://localhost:23100/search/datasets/123/editions/2017/versions/1/dimensions/aggregate?q=term&offset=50", nil)
		w := httptest.NewRecorder()

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
		So(w.Body.String(), ShouldEqual, "the maximum offset has been reached, the offset cannot be more than "+strconv.Itoa(defaultMaxResults)+"\n")
	})

	Convey("Given search API fails to connect to elastic search cluster return status 500 (internal service error)", t, func() {
		r := httptest.NewRequest("GET", "http://localhost:23100/search/datasets/123/editions/2017/versions/1/dimensions/aggregate?q=term", nil)
		w := httptest.NewRecorder()

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{InternalServerError: true}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
		So(w.Body.String(), ShouldEqual, "Failed to process the request due to an internal error\n")
	})

	Convey("Given the search index does not exist return status 404 (not found)", t, func() {
		r := httptest.NewRequest("GET", "http://localhost:23100/search/datasets/123/editions/2017/versions/1/dimensions/aggregate?q=term", nil)
		w := httptest.NewRecorder()

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{IndexNotFound: true}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, "Resource not found\n")
	})
}

func TestAuthenticatedWebUserCannotSeeUnpublished(t *testing.T) {
	Convey("Given web subnet, when an authenticated GET is made, then the dataset api should not see authentication and returns not found, so we return status 404 (not found)", t, func() {
		r := httptest.NewRequest("GET", "http://localhost:23100/search/datasets/123/editions/2017/versions/1/dimensions/aggregate?q=term", nil)
		w := httptest.NewRecorder()
		r.Header.Add("internal-token", secretKey)

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{NotFoundIfAuthBlank: true}, &mocks.Elasticsearch{}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, "Resource not found\n")
	})
}

func TestCreateSearchIndexReturnsOK(t *testing.T) {
	Convey("Given instance and dimension exist return a status 200 (ok)", t, func() {
		r := httptest.NewRequest("PUT", "http://localhost:23100/search/instances/123/dimensions/aggregate", nil)
		w := httptest.NewRecorder()
		r.Header.Add("internal-token", secretKey)

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, true)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusOK)
	})
}

func TestFailToCreateSearchIndex(t *testing.T) {
	Convey("Given a request to create search index but no auth header is set return a status 404 (not found)", t, func() {
		r := httptest.NewRequest("PUT", "http://localhost:23100/search/instances/123/dimensions/aggregate", nil)
		w := httptest.NewRecorder()

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, true)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, "Resource not found\n")
	})

	Convey("Given a request to create search index but the auth header is wrong return a status 404 (not found)", t, func() {
		r := httptest.NewRequest("PUT", "http://localhost:23100/search/instances/123/dimensions/aggregate", nil)
		w := httptest.NewRecorder()
		r.Header.Add("internal-token", "abcdef")

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, true)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, "Resource not found\n")
	})

	Convey("Given a request to create search index but unable to connect to kafka broker return a status 500 (internal service error)", t, func() {
		r := httptest.NewRequest("PUT", "http://localhost:23100/search/instances/123/dimensions/aggregate", nil)
		w := httptest.NewRecorder()
		r.Header.Add("internal-token", secretKey)

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{ReturnError: true}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, true)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
		So(w.Body.String(), ShouldEqual, "Failed to process the request due to an internal error\n")
	})
}

func TestDeleteSearchIndexReturnsOK(t *testing.T) {
	Convey("Given a search index exists return a status 200 (ok)", t, func() {
		r := httptest.NewRequest("DELETE", "http://localhost:23100/search/instances/123/dimensions/aggregate", nil)
		w := httptest.NewRecorder()
		r.Header.Add("internal-token", secretKey)

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, true)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusOK)
	})
}

func TestFailToDeleteSearchIndex(t *testing.T) {
	Convey("Given a search index exists but no auth header set return a status 404 (not found)", t, func() {
		r := httptest.NewRequest("DELETE", "http://localhost:23100/search/instances/123/dimensions/aggregate", nil)
		w := httptest.NewRecorder()

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, true)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, "Resource not found\n")
	})

	Convey("Given a search index exists but auth header is wrong return a status 404 (not found)", t, func() {
		r := httptest.NewRequest("DELETE", "http://localhost:23100/search/instances/123/dimensions/aggregate", nil)
		w := httptest.NewRecorder()
		r.Header.Add("internal-token", "abcdef")

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, true)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, "Resource not found\n")
	})

	Convey("Given a search index exists but unable to connect to elasticsearch cluster return a status 500 (internal service error)", t, func() {
		r := httptest.NewRequest("DELETE", "http://localhost:23100/search/instances/123/dimensions/aggregate", nil)
		w := httptest.NewRecorder()
		r.Header.Add("internal-token", secretKey)

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{InternalServerError: true}, defaultMaxResults, true)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
		So(w.Body.String(), ShouldEqual, "Failed to process the request due to an internal error\n")
	})

	Convey("Given a search index does not exists return a status 404 (not found)", t, func() {
		r := httptest.NewRequest("DELETE", "http://localhost:23100/search/instances/123/dimensions/aggregate", nil)
		w := httptest.NewRecorder()
		r.Header.Add("internal-token", secretKey)

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{IndexNotFound: true}, defaultMaxResults, true)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, "Resource not found\n")
	})
}

func TestCheckhighlights(t *testing.T) {
	Convey("Given the elasticsearch results contain highlights then the correct snippet pairs are returned", t, func() {
		result := models.HitList{
			Highlight: models.Highlight{
				Code:  []string{"\u0001Sstrangeness\u0001E"},
				Label: []string{"04 \u0001SHousing\u0001E, water, \u0001Selectricity\u0001E, gas and other fuels"},
			},
		}
		result = getSnippets(result)
		So(len(result.Source.Matches.Code), ShouldEqual, 1)
		So(result.Source.Matches.Code[0].Start, ShouldEqual, 1)
		So(result.Source.Matches.Code[0].End, ShouldEqual, 11)
		So(len(result.Source.Matches.Label), ShouldEqual, 2)
		So(result.Source.Matches.Label[0].Start, ShouldEqual, 4)
		So(result.Source.Matches.Label[0].End, ShouldEqual, 10)
		So(result.Source.Matches.Label[1].Start, ShouldEqual, 20)
		So(result.Source.Matches.Label[1].End, ShouldEqual, 30)
	})
}

func getSearchResults(body *bytes.Buffer) *models.SearchResults {
	jsonBody, err := ioutil.ReadAll(body)
	if err != nil {
		os.Exit(1)
	}

	searchResults := &models.SearchResults{}
	if err := json.Unmarshal(jsonBody, searchResults); err != nil {
		os.Exit(1)
	}

	return searchResults
}

func TestDeleteEndpointInWebReturnsNotFound(t *testing.T) {
	Convey("Given a search index exists and credentials are correct, return a status 404 (not found)", t, func() {
		r := httptest.NewRequest("DELETE", "http://localhost:23100/search/instances/123/dimensions/aggregate", nil)
		w := httptest.NewRecorder()
		r.Header.Add("internal-token", secretKey)

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, "404 page not found\n")
	})

	Convey("Given a search index exists and credentials are incorrect, return a status 404 (not found)", t, func() {
		r := httptest.NewRequest("DELETE", "http://localhost:23100/search/instances/123/dimensions/aggregate", nil)
		w := httptest.NewRecorder()
		r.Header.Add("internal-token", "not right")

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldContainSubstring, "404 page not found")
	})
}

func TestCreateSearchIndexEndpointInWebReturnsNotFound(t *testing.T) {
	Convey("Given instance and dimension exist and has valid auth return a status 404 (not found)", t, func() {
		r := httptest.NewRequest("PUT", "http://localhost:23100/search/instances/123/dimensions/aggregate", nil)
		w := httptest.NewRecorder()
		r.Header.Add("internal-token", secretKey)

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldContainSubstring, "404 page not found")
	})

	Convey("Given a request to create search index and no private endpoints when a bad auth header is used, return a status 404 (not found)", t, func() {
		r := httptest.NewRequest("PUT", "http://localhost:23100/search/instances/123/dimensions/aggregate", nil)
		w := httptest.NewRecorder()
		r.Header.Add("internal-token", "not right")

		api := routes(host, secretKey, datasetAPISecretKey, mux.NewRouter(), &mocks.BuildSearch{}, &mocks.DatasetAPI{}, &mocks.Elasticsearch{}, defaultMaxResults, false)
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldContainSubstring, "404 page not found")
	})
}
