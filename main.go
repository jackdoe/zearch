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
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	FILENAME_WEIGHT = 200
	FILEPATH_WEIGHT = 1
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
	pdirtoindex := flag.String("dir-to-index", "", "directory to index")
	pstoredir := flag.String("index-store-dir", "/tmp/zearch", "directory to store the index")
	paddr := flag.String("bind", ":8080", "address to bind to")
	flag.Parse()

	if len(*pdirtoindex) > 0 {
		a := strings.Split(*pdirtoindex, ",")
		took(fmt.Sprintf("indexing %#v", a), func() {
			doIndex(*pstoredir, a)
		})
		os.Exit(0)
	}

	index := NewIndex(*pstoredir)
	http.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		if id, err := strconv.Atoi(r.URL.RawQuery); err == nil {
			path, ok := index.fetchForward(id)
			if !ok {
				w.WriteHeader(http.StatusNotFound)
			} else {
				if file, err := ioutil.ReadFile(path); err == nil {
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

		queries := []Query{}
		unescaped, _ := url.QueryUnescape(r.URL.RawQuery)
		tokenize(unescaped, func(text string, weird int) {
			queries = append(queries, NewTerm(text))
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
		index.executeQuery(query, func(id int32, score int64) {
			total++
			add(&Hit{Id: id, Score: score})
		})

		for _, hit := range hits {
			hit.Path, _ = index.fetchForward(int(hit.Id))
		}

		elapsed := time.Since(t0)
		totalfiles, approxterms := index.stats()
		res := &Result{
			Hits:          hits,
			FilesMatching: total,
			FilesInIndex:  totalfiles,
			TokensInIndex: approxterms,
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
</head>
<body>
<a href="https://github.com/jackdoe/zearch"><img style="position: absolute; top: 0; right: 0; border: 0;" src="https://camo.githubusercontent.com/a6677b08c955af8400f44c6298f40e7d19cc5b2d/68747470733a2f2f73332e616d617a6f6e6177732e636f6d2f6769746875622f726962626f6e732f666f726b6d655f72696768745f677261795f3664366436642e706e67" alt="Fork me on GitHub" data-canonical-src="https://s3.amazonaws.com/github/ribbons/forkme_right_gray_6d6d6d.png"></a>
<input id=q autofocus>&nbsp;<small>zearch.io: go linux jdk8 glibc curator hadoop hbase kafka log4j2 lucene-solr mesos musl perl5 spark</small><br>
<pre id=res></pre>
</body>
<script>
var res = document.getElementById("res")
var work = function(query) {
    var s = ""
    res.innerHTML = s

    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/search?' + query);
    xhr.send(null);
    xhr.onreadystatechange = function () {
       if (xhr.readyState === 4) {
           if (xhr.status === 200) {
               data = JSON.parse(xhr.responseText);
               s += "took: " + data.TookSeconds.toFixed(5) + "s, matching: " + data.FilesMatching + ", searched in " + data.FilesInIndex + " files and " + data.TokensInIndex + " tokens\n"
               for (var i = 0; i < data.Hits.length; i++) {
                   var hit = data.Hits[i]
                   s +=  hit.Score + " <a href='/fetch?"+hit.Id +"#"+hit.Path+"'>"+hit.Path+"</a>\n"
               }
               res.innerHTML = s
               window.location.hash = query
           } else {
               res.innerHTML = "error fetching results, status:" + xhr.status + ", text: xhr.responseText"
           }
       }
   }
}

var q = document.getElementById("q")
q.addEventListener('keyup', function(event) { work(q.value) });
q.value = window.location.hash.substr(1)
if (q.value > 0)
        work(q.value)
</script>
</html>`
		fmt.Fprintf(w, s)
	})

	log.Printf("listening on %s\n", *paddr)
	log.Fatal(http.ListenAndServe(*paddr, nil))
}
