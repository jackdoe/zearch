# zearch [ work in progress ]

basic inverted index based code search in ~900 lines.

![screenshot](https://raw.githubusercontent.com/jackdoe/zearch/master/screenshot.gif)

# run

* indexing for the first time

```
2016/01/04 01:57:01 []string{"/SRC/glibc", "/SRC/go", "/SRC/jdk8", "/SRC/linux", "/SRC/musl", "/SRC/perl5"}
2016/01/04 01:57:01 starting indexer: 0/4
2016/01/04 01:57:01 starting indexer: 1/4
2016/01/04 01:57:01 starting indexer: 2/4
2016/01/04 01:57:01 starting indexer: 3/4
2016/01/04 01:57:58 done
2016/01/04 01:57:58 indexing []string{"/SRC/glibc", "/SRC/go", "/SRC/jdk8", "/SRC/linux", "/SRC/musl", "/SRC/perl5"}: 56.653851s
2016/01/04 01:58:24 flushToDisk /tmp/zearch.index.bin: 25.692545s
2016/01/04 01:58:24 indexing is done, start without arguments

```

it will create 4 shards with prefix `/tmp/zearch.index.bin`, each of which is binary dump of string arrays and postings, 
it will mmap them and search in them, indexing takes a more memory since it builds everything in-memory and then dumps it to disk

```
jack@foo ~ $ du -sh /tmp/zearch.index.bin*
984K    /tmp/zearch.index.bin.shard.0.forward.data
268K    /tmp/zearch.index.bin.shard.0.forward.header
4.3M    /tmp/zearch.index.bin.shard.0.inverted.data
7.3M    /tmp/zearch.index.bin.shard.0.inverted.header
6.6M    /tmp/zearch.index.bin.shard.0.postings
968K    /tmp/zearch.index.bin.shard.1.forward.data
264K    /tmp/zearch.index.bin.shard.1.forward.header
4.3M    /tmp/zearch.index.bin.shard.1.inverted.data
7.4M    /tmp/zearch.index.bin.shard.1.inverted.header
6.6M    /tmp/zearch.index.bin.shard.1.postings
968K    /tmp/zearch.index.bin.shard.2.forward.data
268K    /tmp/zearch.index.bin.shard.2.forward.header
4.8M    /tmp/zearch.index.bin.shard.2.inverted.data
8.5M    /tmp/zearch.index.bin.shard.2.inverted.header
7.0M    /tmp/zearch.index.bin.shard.2.postings
964K    /tmp/zearch.index.bin.shard.3.forward.data
264K    /tmp/zearch.index.bin.shard.3.forward.header
4.4M    /tmp/zearch.index.bin.shard.3.inverted.data
7.5M    /tmp/zearch.index.bin.shard.3.inverted.header
6.7M    /tmp/zearch.index.bin.shard.3.postings
```


* if the index is ready

```
$ go run *.go  # without any arguments
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
* text.Scanner + basic alphanumeric filter is used for tokenization, besides basenames everything else must be searched with full token `__realpath` for example cannot be found with `path`
* basenames can be searched with left edge ngrams so, `atomic.go` can be found with `a,at,ato,atom,atomic`, and the weight is increasing as they go closer to the full word
* the max token len is MAX_TOKEN_LEN which is set to 10 characters, this is based on no data at all, just feels ok when i search, not only i dont have to type more than 10 characters to find the thing, but very very few things conflict after 10 characters
* the doc id is id << 10 | weight, so the max weight is 1024 and we can store max 2097152 (2**21) files, otherwise the postinglist has to be moved from `[]int32` to `[]int64`

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
