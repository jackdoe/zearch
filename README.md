# zearch [ work in progress ]

basic inverted index based code search in ~900 lines.

![screenshot](https://raw.githubusercontent.com/jackdoe/zearch/master/screenshot.gif)

fixme: the demo was made when the index was case insensitive, now you have to search for AtomicLong in order to find it

# run

* indexing for the first time

```
$ go build
$ ./zearch /SRC/
2016/01/05 01:10:56 []string{"/SRC/"}
2016/01/05 01:10:56 creating new shard: /tmp/zearch.index.bin/shard.0
2016/01/05 01:10:56 creating new shard: /tmp/zearch.index.bin/shard.1
2016/01/05 01:10:56 creating new shard: /tmp/zearch.index.bin/shard.2
2016/01/05 01:10:56 creating new shard: /tmp/zearch.index.bin/shard.3
2016/01/05 01:11:07 creating new shard: /tmp/zearch.index.bin/shard.4
2016/01/05 01:11:07 creating new shard: /tmp/zearch.index.bin/shard.5
2016/01/05 01:11:07 creating new shard: /tmp/zearch.index.bin/shard.6
2016/01/05 01:11:07 creating new shard: /tmp/zearch.index.bin/shard.7
2016/01/05 01:11:14 creating new shard: /tmp/zearch.index.bin/shard.8
2016/01/05 01:11:14 creating new shard: /tmp/zearch.index.bin/shard.9
2016/01/05 01:11:14 creating new shard: /tmp/zearch.index.bin/shard.10
2016/01/05 01:11:14 creating new shard: /tmp/zearch.index.bin/shard.11
2016/01/05 01:11:23 creating new shard: /tmp/zearch.index.bin/shard.12
2016/01/05 01:11:23 creating new shard: /tmp/zearch.index.bin/shard.13
2016/01/05 01:11:23 creating new shard: /tmp/zearch.index.bin/shard.14
2016/01/05 01:11:23 creating new shard: /tmp/zearch.index.bin/shard.15
2016/01/05 01:11:36 creating new shard: /tmp/zearch.index.bin/shard.16
2016/01/05 01:11:36 creating new shard: /tmp/zearch.index.bin/shard.17
2016/01/05 01:11:36 creating new shard: /tmp/zearch.index.bin/shard.18
2016/01/05 01:11:36 creating new shard: /tmp/zearch.index.bin/shard.19
2016/01/05 01:11:42 done
2016/01/05 01:11:42 indexing []string{"/SRC/"}: 45.813937s
```

it will create N shards (one shard every 15_000 files) with prefix `/tmp/zearch.index.bin/shard.*`, each of which is binary dump of string arrays and postings,
it will mmap them and search in them, indexing takes a more memory since it builds everything in-memory and then dumps it to disk

```
jack@foo ~ $ du -h /tmp/zearch.index.bin
9.5M    /tmp/zearch.index.bin/shard.1
...
9.6M    /tmp/zearch.index.bin/shard.15
245M    /tmp/zearch.index.bin

jack@foo ~ $ du -ah /tmp/zearch.index.bin/shard.0/
2.1M    /tmp/zearch.index.bin/shard.0/inverted.data
4.0M    /tmp/zearch.index.bin/shard.0/inverted.header
180K    /tmp/zearch.index.bin/shard.0/forward.data
3.0M    /tmp/zearch.index.bin/shard.0/posting
68K     /tmp/zearch.index.bin/shard.0/forward.header

```

* when the index is ready

```
$ ./zearch # without any arguments
2016/01/02 13:10:51 listening on port 8080
```

# json api

is just uses the `QUERY_STRING` so searching for `udp ipv4` is `http://localhost:8080/search?udp%20ipv4`

```
$ curl -s 'http://localhost:8080/search?udp%20ipv4' | json_xs
{
   "TookSeconds" : 0.000107539,
   "FilesMatching" : 41,
   "FilesInIndex" : 66023,
   "TokensInIndex" : 635573
   "Hits" : [
      {
         "Path" : "/SRC/linux/net/ipv4/udp.c",
         "Id" : 63337,
         "Score" : 204
      },
      ...
      ...
   ],
}
```

# search

* just open http://localhost:8080, and be amazed by the design :D
* basenames can be searched with left edge ngrams so, `atomic.go` can be found with `a,at,ato,atom,atomic`, and the weight is increasing as they go closer to the full word
* the doc id is id << 10 | weight, so the max weight is 1024 and we can store max 2097152 (2**21) files, otherwise the postinglist has to be moved from `[]int32` to `[]int64`
* index is case sensitive, look at [tokenizer](#tokenizer) for more detail on the tokenizer

# emacs

`zearch-search-current` - searches the current word or marked selection

```
(add-to-list 'load-path "/path/to/zearch.el")
(require 'zearch)
(define-key global-map (kbd "M-s") 'zearch-search-current)

;; i use it with key-chord:
;; (key-chord-define-global "jj" 'zearch-search-current)
```

![screenshot](https://raw.githubusercontent.com/jackdoe/zearch/master/screenshot-emacs.gif)

# tokenizer

extremely basic *case sensitive* tokenizer splits tokens on `(c >= 'a' && c <= 'z') || c == '_' || c == ':' || (c >= '0' && c <= '9')`, and tries to upsort things that have "function|func|class|sub" on their line, except "function\func|class.. etc" they are treated as regular tokens.

```
func tokenize(input string, cb func(string, int)) {
	weird := 0
	start, end := -1, -1
	for i, c := range input {
		if c == '\n' || c == '\r' {
			weird = 0
		}
		if c >= 'A' && c <= 'Z' {
			c |= 0x20
		}
		if (c >= 'a' && c <= 'z') || c == '_' || c == ':' || (c >= '0' && c <= '9') {
			if start == -1 {
				start = i
				end = start
			}
			end++
		} else {
			if end-start > 0 {
				s := input[start:end]
				if _, ok := WEIRD[s]; ok {
					weird = 1
					cb(s, 0)
				} else {
					cb(s, weird)
				}
			}
			start, end = -1, -1
		}
	}
	if end-start > 0 {
		cb(input[start:end], weird)
	}
}
```

# TODO

* store the postinglists in compressed form
* real time indexing
* "fuzzy" 2,3 ngram tokens
* support for queries like "udp -java"
