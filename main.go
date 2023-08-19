package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"index/suffixarray"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/search", Searcher{}.Search)

	fmt.Printf("shakesearch available at http://localhost:%s...", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatal(err)
	}
}

var (
	//go:embed completeworks.txt
	source      []byte
	sourceIndex = suffixarray.New(bytes.ToLower(source))
	pageSize    = 20
)

type Searcher struct{}

func (s Searcher) Search(w http.ResponseWriter, r *http.Request) {
	query := bytes.ToLower([]byte(r.URL.Query().Get("q")))
	if len(query) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing search query in URL params"))
		return
	}
	p, _ := strconv.Atoi(r.URL.Query().Get("p"))

	isHTMX := r.Header.Get(`HX-Request`) == "true"

	results := []string{}
	idxs := sourceIndex.Lookup(query, pageSize*p+pageSize+1) // fetch 1 extra so we know to write load more button
	for n, idx := range idxs[pageSize*p:] {
		if n == pageSize {
			if isHTMX {
				fmt.Fprintf(w, `<tr><td><button id="load-more" hx-get="/search?p=%d&q=%s" hx-target="closest tr" hx-swap="outerHTML">Load more</button></td></tr>`, p+1, query)
			}
			break
		}
		tail := idx + len(query)
		start, middle, end := source[idx:idx], source[idx:tail], source[tail:tail]
		for expand := 0; expand < 1000 && len(start)+len(middle)+len(end) < 300; expand++ {
			if cursor := tail + expand; len(end) <= len(start) && (cursor >= len(source)-1 || source[cursor] == '.' || source[cursor+1] == '\n') {
				end = source[tail : cursor+1]
			}
			if cursor := idx - expand; len(start) <= len(end) && (cursor <= 1 || source[cursor] == '\n' || source[cursor-1] == '.' && source[cursor] == ' ') {
				start = source[cursor:idx]
			}
		}
		startStr, middleStr, endStr := html.EscapeString(string(start)), html.EscapeString(string(middle)), html.EscapeString(string(end))

		if isHTMX {
			fmt.Fprintf(w, `<tr><td><pre>%s<mark>%s</mark>%s</pre></td></tr>`, startStr, middleStr, endStr)
			continue
		}
		results = append(results, startStr+middleStr+endStr)
	}
	if !isHTMX {
		log.Printf("%d results for api query %s", len(results), query)
		json.NewEncoder(w).Encode(results)
	}
}

func (s Searcher) Load(file string) error {
	// Load left in for test compatability, using embed is much simpler
	return nil
}

func handleSearch(s Searcher) http.HandlerFunc {
	// handleSearch left in for test compatability
	return s.Search
}
