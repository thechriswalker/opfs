package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

const (
	DEFAULT_PAGE_SIZE = 30
	THUMBNAIL_SMALL   = 200
	THUMBNAIL_LARGE   = 1024
	BUFFER_POOL_SIZE  = 10
	BUFFER_BYTE_SIZE  = 1024 * 100
)

//The API returns a http.Handler which can serve json requests
//using the Service given.
type Api struct {
	listen     string
	service    *Service
	mux        http.Handler
	headers    map[string]string //common header to set on every response
	showErrors bool
	ui         http.Handler
}

//to satisfy the http.Handler interface
func (a *Api) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//figure out what is being requested and with which method.
	a.mux.ServeHTTP(w, r)
}

func (a *Api) httpWrapJSON(f func(a *Api, r *http.Request) (int, interface{}, error)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s, v, err := f(a, r); err != nil {
			a.serveError(w, err)
		} else {
			a.serveJSON(w, s, v)
		}
	})
}

//ensures we have a nice JSON version of the error before we
//send it on. also it logs the error and removes the "real" error
//from the JSON unless we have specifically configured it not to.
func (a *Api) serveError(w http.ResponseWriter, err error) {
	e := NewApiError(err, "Unknown Error", 500)
	log.Println(e)
	if !a.showErrors {
		//kill the error, before serving.
		e.Err = ""
	}
	a.serveJSON(w, e.Code, e)
}

//A buffer pool for json encoding
var bufferPool = NewBufferPool(BUFFER_POOL_SIZE, BUFFER_BYTE_SIZE) //10 x 100k buffers

//Serve a JSON response
func (a *Api) serveJSON(w http.ResponseWriter, s int, v interface{}) {
	setCommonHeaders(a, w)
	if s/100 == 3 {
		//redirection!
		w.Header().Set("Location", v.(string))
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(s)
		return
	}
	if v != nil {
		b := bufferPool.Get()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(b).Encode(v)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", b.Len()))
		w.WriteHeader(s)
		b.WriteTo(w)
		bufferPool.Recycle(b)
	} else {
		//no body, e.g. 204
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(s)
	}
}

func (a *Api) dataRetrieveById(w http.ResponseWriter, r *http.Request) {
	hash := mux.Vars(r)["item"]
	rsc, err := a.service.store.Get(hash)
	if err != nil {
		a.serveError(w, err)
	} else {
		meta, err := a.service.store.Meta(hash)
		if err != nil {
			a.serveError(w, err)
		}
		w.Header().Set("Content-Type", meta.Mime)
		w.Header().Set("ETag", meta.Hash) //the hash is immutable.
		http.ServeContent(w, r, "", meta.Added, rsc)
	}
}

func (a *Api) thumbSmall(w http.ResponseWriter, r *http.Request) {
	thumbnail(a, w, r, THUMBNAIL_SMALL)
}

func (a *Api) thumbLarge(w http.ResponseWriter, r *http.Request) {
	thumbnail(a, w, r, THUMBNAIL_LARGE)
}

func thumbnail(a *Api, w http.ResponseWriter, r *http.Request, size int) {
	hash := mux.Vars(r)["item"]
	meta, err := a.service.store.Meta(hash)
	if err != nil {
		a.serveError(w, err)
		return
	}
	rs, mime, err := getItemThumbnail(a.service, meta, size)
	if err != nil {
		a.serveError(w, err)
		return
	}
	//if this is closeable we should close it.
	if closer, ok := rs.(io.Closer); ok {
		defer closer.Close()
	}
	if mime != "" {
		w.Header().Set("Content-Type", mime)
	}
	if closer, ok := rs.(io.Closer); ok {
		defer closer.Close()
	}
	http.ServeContent(w, r, "", meta.Added, rs)
}

func generateGif(rw http.ResponseWriter, r *http.Request) {
	width, height := mux.Vars(r)["width"], mux.Vars(r)["height"]
	w, errw := strconv.ParseUint(width, 10, 16)
	h, errh := strconv.ParseUint(height, 10, 16)
	if errw != nil || errh != nil {
		w, h = 1, 1
	}
	rw.Header().Set("Content-Type", "image/gif")
	http.ServeContent(rw, r, "", time.Now(), bytes.NewReader(createGif(uint16(w), uint16(h))))
}

//Api error for simple error handling
type ApiError struct {
	Code    int    //this is used for setting status code as well as
	Err     string `json:"Error,omitempty"` //the real error (for logging/debug)
	Message string //the user-fronting version of the error
}

var (
	apiDefaultError   = []byte(`{"Code":500,"Error":"Internal Server Error"}`)
	errNotImplemented = &ApiError{
		Code:    500,
		Message: "API Method not Implemented",
	}
)

//satisfy the error interface
func (a *ApiError) Error() string {
	return fmt.Sprintf("API Error [%d](%s) %s", a.Code, a.Err, a.Message)
}

//create/ensure an error is an API error instance
func NewApiError(err error, msg string, code int) *ApiError {
	if e, ok := err.(*ApiError); ok {
		return e
	}
	return &ApiError{Code: code, Err: err.Error(), Message: msg}
}

//sets headers common to all requests
func setCommonHeaders(a *Api, w http.ResponseWriter) {
	if a.headers != nil {
		headers := w.Header()
		for k, v := range a.headers {
			headers.Set(k, v)
		}
	}
}

//This is the one that sets up all the routes.
func (a *Api) initRoutes() {
	//mux...
	router := mux.NewRouter()
	router.Handle("/api", a.httpWrapJSON(apiInfo)).Methods("GET", "HEAD")
	router.Handle("/api/gif/{width}/{height}", http.HandlerFunc(generateGif)).Methods("GET", "HEAD")
	router.Handle("/api/tags/add", a.httpWrapJSON(apiAddTagItems)).Methods("POST")
	router.Handle("/api/tags/remove", a.httpWrapJSON(apiRemoveTagItems)).Methods("POST")
	router.Handle("/api/search", a.httpWrapJSON(apiSearch)).Methods("GET", "HEAD")
	router.Handle("/api/items/{item:sha1-[0-9a-f]{40}}", a.httpWrapJSON(apiGetItem)).Methods("GET", "HEAD")
	router.Handle("/api/items/{item:sha1-[0-9a-f]{40}}", a.httpWrapJSON(apiDeleteItem)).Methods("DELETE")
	router.Handle("/api/items/{item:sha1-[0-9a-f]{40}}/tags", a.httpWrapJSON(apiSetItemTags)).Methods("POST", "HEAD")
	router.Handle("/api/items/{item:sha1-[0-9a-f]{40}}/description", a.httpWrapJSON(apiSetItemDescription)).Methods("POST")
	router.Handle("/api/items/{item:sha1-[0-9a-f]{40}}/raw", http.HandlerFunc(a.dataRetrieveById)).Methods("GET", "HEAD")
	router.Handle("/api/items/{item:sha1-[0-9a-f]{40}}/thumb/small", http.HandlerFunc(a.thumbSmall)).Methods("GET", "HEAD")
	router.Handle("/api/items/{item:sha1-[0-9a-f]{40}}/thumb/large", http.HandlerFunc(a.thumbLarge)).Methods("GET", "HEAD")
	router.PathPrefix("/").Handler(a.ui).Methods("GET", "HEAD")

	a.mux = router
}

type ApiItemsResponse struct {
	Total, Count int    //total is all results, count is number returned
	Next         string `json:",omitempty"`
	Results      []json.RawMessage
}

func toNextUri(r *http.Request, p *Pagination) string {
	if p == nil {
		return ""
	}
	//get URL, replace Query params, from/size
	u := r.URL
	q := u.Query()
	q.Set("from", fmt.Sprintf("%d", p.From))
	q.Set("size", fmt.Sprintf("%d", p.Size))
	u.RawQuery = q.Encode()
	return u.String()
}

func searchToResponse(r *http.Request, s *SearchResult) *ApiItemsResponse {
	return &ApiItemsResponse{
		Total:   s.Count,
		Count:   len(s.Results),
		Next:    toNextUri(r, s.Next),
		Results: s.Results,
	}
}

func getPagination(q url.Values, defaultSize int) (p *Pagination) {
	var err error
	p = &Pagination{}
	if p.From, err = strconv.Atoi(q.Get("from")); err != nil {
		p.From = 0
	}
	if p.Size, err = strconv.Atoi(q.Get("size")); err != nil {
		p.Size = defaultSize
	}
	return
}

//Api functions.

//get server info
func apiInfo(a *Api, r *http.Request) (int, interface{}, error) {
	return 200, map[string]string{"Name": "OPFS", "Version": a.service.Version()}, nil
}

type TagRequest struct {
	Tag   string   //the tag to add/remove
	Items []string //list of items to tag.
}

func apiAddTagItems(a *Api, r *http.Request) (int, interface{}, error) {
	//this should add the tag to the give item ids.
	//payload should be JSON {"Items":[]}
	return 0, nil, errNotImplemented
}

func apiRemoveTagItems(a *Api, r *http.Request) (int, interface{}, error) {
	//this should remove the tag from the give item ids.
	//payload should be JSON {"Items":[]}
	return 0, nil, errNotImplemented
}

type nearQuery struct {
	L *LatLon //location
	R int     //radius
}

type apiSearchParams struct {
	Types []ItemType
	Match map[string]interface{}
	Range map[string][2]interface{}
	Near  *nearQuery
	Sort  map[string]SortDir
	Tags  []string
	Text  string
}

//This is the big one, almost all requests are going to come through here.
func apiSearch(a *Api, r *http.Request) (int, interface{}, error) {
	/*
	   query: {"Types":["Video"],
	    "Match":{"field":"value"},
	    "Range":{"field":[min,max]},
	    "Near":{L:"0.23241,0.234235",R:5},
	    "Sort":{field:dir},
	    "Tags":["tags","to","match"]
	    "Text":"full text against description"
	   }
	   from: int,
	   size: int,
	*/
	qs := r.URL.Query()
	params := &apiSearchParams{}
	err := json.Unmarshal([]byte(qs.Get("query")), params)
	log.Println("?query=", qs.Get("query"))
	if err != nil {
		return 0, nil, NewApiError(err, "Could not parse ?query", 400)
	}
	//now construct the Search Query
	search := a.service.indexer.NewQuery()
	if params.Types != nil && len(params.Types) > 0 {
		log.Println(params.Types)
		search.Type(params.Types...)
	}
	explicitDeletedMatch := false
	if params.Match != nil && len(params.Match) > 0 {
		for field, val := range params.Match {
			search.Match(field, val)
			if field == "Deleted" {
				explicitDeletedMatch = true
			}
		}
	}

	if params.Range != nil && len(params.Range) > 0 {
		for field, val := range params.Range {
			search.Range(field, val[0], val[1])
			if field == "Deleted" {
				explicitDeletedMatch = true
			}
		}
	}

	if explicitDeletedMatch {
		//they did do something with deleted, so we'd better allow deleted results.
		search.AllowDeleted(true)
	}

	if params.Near != nil {
		search.Near(params.Near.L, params.Near.R)
	}
	if params.Sort != nil {
		for field, dir := range params.Sort {
			search.Sort(field, dir)
			break //only the first one!
		}
	}
	if params.Tags != nil && len(params.Tags) > 0 {
		search.Tagged(params.Tags...)
	}
	//no text search yet...
	//@TODO text search
	page := getPagination(qs, DEFAULT_PAGE_SIZE)

	//now query!
	res, err := a.service.indexer.Search(search, page)
	if err != nil {
		return 0, nil, err
	}
	return 200, searchToResponse(r, res), nil
}

func apiGetItem(a *Api, r *http.Request) (int, interface{}, error) {
	hash := mux.Vars(r)["item"]
	item, err := a.service.store.Meta(hash)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil, NewApiError(err, fmt.Sprintf("No such item: %s", hash), 404)
		} else {
			return 0, nil, err
		}
	}
	deletedOK := false
	if qDeleted := r.URL.Query().Get("deleted"); qDeleted == "true" {
		deletedOK = true
	}
	//check deleted.
	if !deletedOK && !item.Deleted.IsZero() {
		return 0, nil, NewApiError(fmt.Errorf("Item is Deleted: %s", hash), fmt.Sprintf("No such item: %s", hash), 404)
	}
	return 200, item, nil
}

func apiDeleteItem(a *Api, r *http.Request) (int, interface{}, error) {
	hash := mux.Vars(r)["item"]
	item, err := a.service.store.Meta(hash)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil, NewApiError(err, fmt.Sprintf("No such item: %s", hash), 404)
		} else {
			return 0, nil, err
		}
	}
	//check deleted.
	if !item.Deleted.IsZero() {
		return 0, nil, NewApiError(fmt.Errorf("Item is Deleted: %s", hash), fmt.Sprintf("No such item: %s", hash), 404)
	}
	//update item!
	//remember deleted is a Pointer to time.Time
	t := time.Now()
	item.Deleted = &t
	if err = item_update(a.service, item); err != nil {
		return 0, nil, NewApiError(err, fmt.Sprintf("Failed to delete item: %s", hash), 500)
	}
	return 204, nil, nil

}

type ItemTagRequest struct {
	Tags []string //list of tags to set on the item.
}

func apiSetItemTags(a *Api, r *http.Request) (int, interface{}, error) {
	//this should SET the given tags to the item, removing others.
	//payload should be JSON {"Tags":[]}
	hash := mux.Vars(r)["item"]
	item, err := a.service.store.Meta(hash)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil, NewApiError(err, fmt.Sprintf("No such item: %s", hash), 404)
		} else {
			return 0, nil, err
		}
	}
	//check deleted.
	if !item.Deleted.IsZero() {
		return 0, nil, NewApiError(fmt.Errorf("Item is Deleted: %s", hash), fmt.Sprintf("No such item: %s", hash), 404)
	}

	tags := &ItemTagRequest{}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(tags); err != nil {
		return 0, nil, NewApiError(err, "Invalid Request Body", 400)
	}
	if tags.Tags == nil {
		tags.Tags = emptySlice
	}
	if err := SetItemTags(a.service, item, tags.Tags...); err != nil {
		return 0, nil, NewApiError(err, "Error tagging Item", 500)
	}
	return 204, nil, nil
}

type ItemDescriptionRequest struct {
	Description string
}

func apiSetItemDescription(a *Api, r *http.Request) (int, interface{}, error) {
	//this should set the item description as given
	//payload should be JSON {"Description":"<string>"}
	hash := mux.Vars(r)["item"]
	item, err := a.service.store.Meta(hash)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil, NewApiError(err, fmt.Sprintf("No such item: %s", hash), 404)
		} else {
			return 0, nil, err
		}
	}
	//check deleted.
	if !item.Deleted.IsZero() {
		return 0, nil, NewApiError(fmt.Errorf("Item is Deleted: %s", hash), fmt.Sprintf("No such item: %s", hash), 404)
	}
	payload := &ItemDescriptionRequest{}
	if err := json.NewDecoder(r.Body).Decode(payload); err != nil {
		return 0, nil, NewApiError(err, "Invalid Request Body", 400)
	}
	item.Description = payload.Description
	if err = item_update(a.service, item); err != nil {
		return 0, nil, NewApiError(err, fmt.Sprintf("Failed to update item: %s", hash), 500)
	}
	return 204, nil, nil
}
