package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"strconv"
	"strings"
	"text/scanner"
	"time"
)

const (
	PORT            = 8080
	FILENAME_WEIGHT = 200
	FILEPATH_WEIGHT = 1
	STORED_INDEX    = "/tmp/index.msgpack.lz4"

	// for me it seems like all tokens > 10 symbols just waste space
	// but depending on your files, this might not be the case
	MAX_TOKEN_LEN = 10
)

type Hit struct {
	Path  string
	Id    int32
	Score int64
}

type Result struct {
	Hits          []*Hit
	FilesMatching int
	FilesInIndex  int
	TokensInIndex int
	TookSeconds   float64
}

func main() {
	flag.Parse()
	args := flag.Args()
	index := NewIndex()
	if len(args) > 0 {
		took(fmt.Sprintf("indexing %#v", args), func() {
			index.doIndex(args)
		})
		took(fmt.Sprintf("dumpToDisk %s", STORED_INDEX), func() {
			index.dumpToDisk(STORED_INDEX)
		})
	} else {
		took(fmt.Sprintf("load %s", STORED_INDEX), func() {
			index.load(STORED_INDEX)
		})
	}

	http.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		if id, err := strconv.Atoi(r.URL.RawQuery); err == nil {
			if id < 0 || id > len(index.Forward)-1 {
				w.WriteHeader(http.StatusNotFound)
			} else {
				if file, err := ioutil.ReadFile(index.Forward[id]); err == nil {
					w.Header().Set("Content-Type", "text/plain")
					w.Write(file)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		t0 := time.Now()

		s := &scanner.Scanner{}
		unescaped, _ := url.QueryUnescape(r.URL.RawQuery)
		initScanner(s, strings.NewReader(unescaped))

		queries := []Query{}
		tokenize(s, func(text string) {
			queries = append(queries, NewTerm(index.postingList(text)))
		})
		var query Query
		if len(queries) == 1 {
			query = queries[0]
		} else {
			query = NewBoolAndQuery(queries)
		}

		hits := []*Hit{}
		maxSize := 100
		add := func(h *Hit) {
			do_insert := false
			if len(hits) < maxSize {
				hits = append(hits, h)
				do_insert = true
			} else if hits[len(hits)-1].Score < h.Score {
				do_insert = true
			}
			if do_insert {
				for i := 0; i < len(hits); i++ {
					if hits[i].Score < h.Score {
						copy(hits[i+1:], hits[i:])
						hits[i] = h
						break
					}
				}
			}
		}

		total := 0
		for query.Next() != NO_MORE {
			score := query.Score()
			total++
			id := query.GetDocId()
			add(&Hit{
				Path:  index.Forward[id],
				Id:    id,
				Score: score,
			})
		}
		elapsed := time.Since(t0)

		res := &Result{
			Hits:          hits,
			FilesMatching: total,
			FilesInIndex:  len(index.Forward),
			TokensInIndex: len(index.Inverted),
			TookSeconds:   elapsed.Seconds(),
		}

		b, err := json.Marshal(res)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		s := `
<html><head><title>zearch</title>
<script src="https://ajax.googleapis.com/ajax/libs/jquery/2.1.4/jquery.min.js"></script>
<script>
$(document).ready(function() {
   var work = function(q) {
        $('#res').html("")

        s = ""
        $.get('/search?' + q, function(data) {
            s += "took: " + data.TookSeconds.toFixed(5) + "s, matching: " + data.FilesMatching + ", searched in " + data.FilesInIndex + " files and " + data.TokensInIndex + " tokens\n"
            for (var i = 0; i < data.Hits.length; i++) {
                var hit = data.Hits[i]
                s +=  hit.Score + " <a href='/fetch?"+hit.Id +"#"+hit.Path+"'>"+hit.Path+"</a>\n"
            }
            $('#res').html(s)
            window.location.hash = q
        })
    }
    $("#q").keyup(function() { work($(this).val()); })
    $("#q").val(window.location.hash.substr(1))
    work($("#q").val())
})
</script></head>
<body>
<input id=q autofocus><br>
<pre id=res></pre>
</body>
</html>`
		fmt.Fprintf(w, s)
	})

	log.Printf("listening on port %d\n", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil))
}
