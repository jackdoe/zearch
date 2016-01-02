# zearch

basic inverted index based code search in ~600 lines.

![screenshot](https://raw.githubusercontent.com/jackdoe/zearch/master/screenshot.gif)

# run

* indexing for the first time

```
2016/01/02 13:03:45 []string{"/SRC/glibc", "/SRC/go", "/SRC/jdk8", "/SRC/linux", "/SRC/musl", "/SRC/perl5"}
2016/01/02 13:03:45 starting indexer: 0/4
2016/01/02 13:03:45 starting indexer: 1/4
2016/01/02 13:03:45 starting indexer: 2/4
2016/01/02 13:03:45 starting indexer: 3/4
2016/01/02 13:04:23 indexing []string{"/SRC/glibc", "/SRC/go", "/SRC/jdk8", "/SRC/linux", "/SRC/musl", "/SRC/perl5"}: 37.412563s
2016/01/02 13:04:29 dumpToDisk /tmp/index.msgpack.lz4: 5.952573s
2016/01/02 13:04:29 listening on port 8080
```

The index is msgpack encoded + lz4 compressed, and having the inverted + forward index of those source trees is about 41mb

```
$ du -ah /tmp/index.msgpack.lz4
41M     /tmp/index.msgpack.lz4
```

The whole index is kept in ram, above takes about 300mb for `66023 files` and `1340808 tokens` while indexing linux kernel, glibc, perl5, and jdk8

* if the index is ready

```
$ go run *.go  # without any arguments
2016/01/02 13:10:51 load /tmp/index.msgpack.lz4: 3.296206s
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

# TODO

* store it off heap
* store the postinglists in compressed form
* real time indexing
* "fuzzy" 2,3 ngram tokens
* support for queries like "udp -java"
* emacs plugin
