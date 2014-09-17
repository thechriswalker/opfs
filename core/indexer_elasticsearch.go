package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const (
	esDEFAULT_BASEURL = "http://127.0.0.1:9200"
	esDEFAULT_INDEX   = "opfs"
)

func init() {
	RegisterIndexer("es-indexer", func(conf *ServiceConfig, s *Service) (Indexer, error) {
		baseurl, ok := conf.Conf["base"]
		baseurl_string := esDEFAULT_BASEURL
		if ok {
			baseurl_string, ok = baseurl.(string)
			if !ok {
				return nil, fmt.Errorf("ES-Indexer Config `base` must be a string value: got `%v`", baseurl)
			}
		}

		index, ok := conf.Conf["index"]
		index_string := esDEFAULT_INDEX
		if ok {
			index_string, ok = baseurl.(string)
			if !ok {
				return nil, fmt.Errorf("ES-Indexer Config `index` must be a string value: got `%v`", index)
			}
		}
		return NewESIndexer(baseurl_string, index_string, nil), nil
	})
}

type ElasticsearchIndexer struct {
	client *http.Client //nil means use default
	base   string       //the baseurl e.g. http://localhost:9200/index (no trailing slash)
	index  string       //which es index to use (maybe this should be dynamic...)
}

var _ Indexer = (*ElasticsearchIndexer)(nil)

func NewESIndexer(base, index string, client *http.Client) *ElasticsearchIndexer {
	if client == nil {
		client = http.DefaultClient
	}
	indexer := &ElasticsearchIndexer{
		client: client,
		base:   base,
		index:  index,
	}
	return indexer
}

func (es *ElasticsearchIndexer) Index(i *Item) error {
	uri := fmt.Sprintf("%s/%s/Item/%s", es.base, es.index, i.Hash)
	msg, err := json.Marshal(i)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", uri, bytes.NewReader(msg))
	if err != nil {
		return err
	}
	resp, err := es.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		log.Println("ES Response:")
		io.Copy(os.Stdout, resp.Body)
		return fmt.Errorf("Bad Response Status Code from ES: %s", resp.Status)
	}
	return nil
}

func (es *ElasticsearchIndexer) Search(query Query, pagination *Pagination) (*SearchResult, error) {
	if pagination == nil {
		pagination = &Pagination{From: 0, Size: 10}
	}
	var next *Pagination
	req := createSearchRequest(es, query, pagination)
	resp, err := es.client.Do(req)
	if err != nil {
		return nil, err
	}
	//read result into es result struct
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("ES status code: %s", resp.Status)
	}
	dec := json.NewDecoder(resp.Body)
	res := &esSearchResult{}
	err = dec.Decode(&res)
	if err != nil {
		return nil, err
	}
	// log.Println(resp.StatusCode)
	raw := make([]json.RawMessage, len(res.Hits.Hits))
	for idx, hit := range res.Hits.Hits {
		raw[idx] = hit.Source
	}
	fromNext := pagination.From + pagination.Size
	if res.Hits.Total > fromNext {
		next = &Pagination{From: fromNext, Size: pagination.Size}
	}
	return &SearchResult{
		Count:   res.Hits.Total,
		Page:    pagination,
		Next:    next,
		Results: raw,
	}, nil
}

func (es *ElasticsearchIndexer) NewQuery() Query {
	return &esQuery{
		filters: []json.RawMessage{},
		sorts:   []json.RawMessage{},
	}
}

var facetTagQuery = []byte(`{"query":{"match_all":{}},"size":0,"facets":{"Tags":{"terms":{"field":"Tags","size":100}}}}`)

func (es *ElasticsearchIndexer) ListTags() ([]string, error) {
	uri := fmt.Sprintf("%s/%s/_search", es.base, es.index)
	req, _ := http.NewRequest("POST", uri, bytes.NewReader(facetTagQuery))
	resp, err := es.client.Do(req)
	if err != nil {
		return nil, err
	}
	//read result into es result struct
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("ES status code: %s", resp.Status)
	}
	dec := json.NewDecoder(resp.Body)
	res := &esSearchResult{}
	err = dec.Decode(&res)
	if err != nil {
		return nil, err
	}
	tags := make([]string, len(res.Facets["Tags"].Terms))
	for i, term := range res.Facets["Tags"].Terms {
		tags[i] = term.Term
	}
	return tags, nil
}

/*{
  "took" : 2,
  "timed_out" : false,
  "_shards" : {
    "total" : 1,
    "successful" : 1,
    "failed" : 0
  },
  "hits" : {
    "total" : 1,
    "max_score" : 1.0,
    "hits" : [ {
      "_index" : "test",
      "_type" : "type1",
      "_id" : "abc",
      "_score" : 1.0, "_source" : {"item":"abc","is":1337}
    } ]
  },
  "facets": {
        "Tags": {
            "_type": "terms",
            "missing": 285,
            "total": 2,
            "other": 0,
            "terms": [
                {
                    "term": "tag:chris",
                    "count": 1
                },
                {
                    "term": "album:demo",
                    "count": 1
                }
            ]
        }
    }
}*/
type esSearchResult struct {
	Took     int                       `json:"took"`
	TimedOut bool                      `json:"timed_out"`
	Shards   map[string]int            `json:"_shards"`
	Hits     *esSearchResultHits       `json:"hits"`
	Facets   map[string]*esFacetResult `json:"facets,omitempty"`
}

type esFacetResult struct {
	Terms []*esFacetTermResult `json:"terms,omitempty"`
}

type esFacetTermResult struct {
	Term  string `json:"term"`
	Count int    `json:"count"`
}
type esSearchResultHits struct {
	Total    int                  `json:"total"`
	MaxScore float64              `json:"max_score"`
	Hits     []*esSearchResultHit `json:"hits"`
}
type esSearchResultHit struct {
	Id     string          `json:"_id"`
	Score  float64         `json:"_score"`
	Source json.RawMessage `json:"_source"`
}

func createSearchRequest(es *ElasticsearchIndexer, query Query, page *Pagination) *http.Request {
	uri := fmt.Sprintf("%s/%s/_search?from=%d&size=%d", es.base, es.index, page.From, page.Size)
	body, _ := json.Marshal(query)
	req, _ := http.NewRequest("POST", uri, bytes.NewReader(body))
	return req
}

type esSearchRequest struct {
	Query map[string]map[string]string `json:"query"`
	Sort  map[string]string            `json:"sort"`
	From  int                          `json:"from"`
	Size  int                          `json:"size"`
}

//the es query is so variable in schema, that is is best to represent like this
type esQuery struct {
	filters      []json.RawMessage
	sorts        []json.RawMessage
	allowDeleted bool
}

//@TODO: this implementation only allows you to call this once!
func (q *esQuery) Type(t ...ItemType) Query {
	//this is a terms filter...
	filter := map[string]map[string][]ItemType{
		"terms": map[string][]ItemType{
			"Type": t,
		},
	}
	raw, _ := json.Marshal(filter)
	q.filters = append(q.filters, raw)
	return Query(q)
}

func (q *esQuery) AllowDeleted(allow bool) Query {
	q.allowDeleted = allow
	return Query(q)
}

//@TODO: this implementation only allows you to call this for a given field once!
func (q *esQuery) Match(field string, val ...interface{}) Query {
	for _, v := range val {
		filter := map[string]map[string]interface{}{"term": make(map[string]interface{}, 1)}
		filter["term"][field] = v
		raw, _ := json.Marshal(filter)
		q.filters = append(q.filters, raw)
	}
	return Query(q)
}

//@TODO: this implementation only allows you to call this once!
func (q *esQuery) Tagged(tags ...string) Query {
	args := make([]interface{}, len(tags))
	for i, t := range tags {
		args[i] = t
	}
	q.Match("Tags", args...)
	return Query(q)
}

//@TODO: this implementation only allows you to call this for a given field once!
func (q *esQuery) Range(field string, min, max interface{}) Query {
	filter := map[string]map[string]map[string]interface{}{"range": make(map[string]map[string]interface{}, 1)}
	filter["range"][field] = make(map[string]interface{})
	if min != nil {
		filter["range"][field]["gte"] = min
	}
	if max != nil {
		filter["range"][field]["lte"] = max
	}
	raw, _ := json.Marshal(filter)
	q.filters = append(q.filters, raw)
	return Query(q)
}

//@TODO: this implementation only allows you to call this once!
func (q *esQuery) Near(pos *LatLon, radiusKm int) Query {
	filter := map[string]map[string]interface{}{
		"geo_distance": map[string]interface{}{
			"distance": fmt.Sprintf("%dkm", radiusKm),
			"Location": pos,
		},
	}
	raw, _ := json.Marshal(filter)
	q.filters = append(q.filters, raw)
	return Query(q)
}

//@TODO: this implementation only allows you to call this once!
func (q *esQuery) Sort(field string, dir SortDir) Query {
	sort := make(map[string]string, 1)
	if dir == SortDirAscending {
		sort[field] = "asc"
	} else {
		sort[field] = "desc"
	}
	raw, _ := json.Marshal(sort)
	q.sorts = append(q.sorts, raw)
	return Query(q)
}

func (q *esQuery) MarshalJSON() ([]byte, error) {
	var buff bytes.Buffer
	//buff.Grow(queryWrapLength + len(filters) + len(sorts))
	if q.allowDeleted == false {
		q.filters = append(q.filters, json.RawMessage([]byte(`{"missing":{"field":"Deleted","existence":true,"null_value":true}}`)))
	}

	buff.Write(queryWrapPrefix)
	if len(q.filters) > 0 {
		filters, _ := json.Marshal(q.filters)
		buff.Write(queryWrapFilterPrefix)
		buff.Write(filters)
		buff.Write(queryWrapObjectClose)
	}
	buff.Write(queryWrapObjectClose)
	buff.Write(queryWrapObjectClose)

	if len(q.sorts) > 0 {
		sorts, _ := json.Marshal(q.sorts)

		buff.Write(queryWrapSortPrefix)
		buff.Write(sorts)
	}
	buff.Write(queryWrapObjectClose)

	log.Println(buff.String())
	return buff.Bytes(), nil
}

var (
	//we or use filters, so our best query is wrapped filtered query
	queryWrapPrefix       = []byte(`{"query":{"filtered":{"query":{"match_all":{}}`)
	queryWrapFilterPrefix = []byte(`,"filter":{"and":`)
	queryWrapSortPrefix   = []byte(`,"sort":`)
	queryWrapObjectClose  = []byte(`}`)
)
