package main

import (
	idx "./index"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Hit struct {
	Path    string
	Id      int32
	Segment int
	Score   int64
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
	pstoredir := flag.String("dir-to-store", path.Join("tmp", "zearch"), "directory to store the index")
	paddr := flag.String("bind", ":8080", "address to bind to")
	flag.Parse()

	rwlock := &sync.RWMutex{}
	if len(*pdirtoindex) > 0 {
		a := strings.Split(*pdirtoindex, ",")
		idx.Took(fmt.Sprintf("indexing %#v", a), func() {
			idx.DoIndex(*pstoredir, a)
		})
		os.Exit(0)
	}
	index := idx.NewIndex(*pstoredir)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for range c {
			rwlock.Lock()

			index.Close()
			index = idx.NewIndex(*pstoredir)

			rwlock.Unlock()
		}
	}()

	http.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		rwlock.RLock()
		defer rwlock.RUnlock()
		splitted := strings.Split(r.URL.RawQuery, ",")
		if len(splitted) != 2 {
			w.WriteHeader(http.StatusNotFound)
		} else {
			id, errId := strconv.Atoi(splitted[0])
			segment, errSegment := strconv.Atoi(splitted[1])
			if errId != nil || errSegment != nil {
				w.WriteHeader(http.StatusBadRequest)
			} else {
				path, ok := index.FetchForward(id, int(segment))
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
			}
		}
	})

	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		t0 := time.Now()

		rwlock.RLock()
		defer rwlock.RUnlock()

		queries := []idx.Query{}
		unescaped, _ := url.QueryUnescape(r.URL.RawQuery)
		idx.Tokenize(unescaped, func(text string, weird int) {
			queries = append(queries, idx.NewTerm(text))
		})
		var query idx.Query
		if len(queries) == 1 {
			query = queries[0]
		} else {
			query = idx.NewBoolAndQuery(queries)
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
		index.ExecuteQuery(query, func(id int32, segment int, score int64) {
			total++
			add(&Hit{Id: id, Segment: segment, Score: score})
		})

		for _, hit := range hits {
			hit.Path, _ = index.FetchForward(int(hit.Id), hit.Segment)
		}

		elapsed := time.Since(t0)
		totalfiles, approxterms := index.Stats()
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
<html>
<head><title>zearch</title>
<style>
body {
    font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif;
    padding: 10px;
}

input {
    width: 200px;
    height: 30px;
    font-size: 18px;
}
</style>
</head>
<body>
<input id=q autofocus><br><small><a href="https://github.com/jackdoe/zearch">zearch.io</a>: go linux freebsd kubernetes cassandra jdk8 glibc curator hadoop hbase kafka log4j2 lucene-solr mesos musl perl5 spark</small><br>
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
                   s +=  hit.Score + " <a href='/fetch?"+hit.Id +"," + hit.Segment + "#" + hit.Path+"'>"+hit.Path+"</a>\n"
               }
               res.innerHTML = s
               window.location.hash = query
           } else {
               res.innerHTML = "error fetching results, status:" + xhr.status + ", text: " + xhr.responseText
           }
       }
   }
}

var q = document.getElementById("q")
q.addEventListener('keyup', function(event) { work(q.value) });
q.value = window.location.hash.substr(1)
if (q.value.length > 0)
        work(q.value)
</script>
</html>`
		fmt.Fprintf(w, s)
	})

	log.Printf("listening on %s\n", *paddr)
	log.Fatal(http.ListenAndServe(*paddr, nil))
}
