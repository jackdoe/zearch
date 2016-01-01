# zearch

basic inverted index based code search in ~600 lines.

![screenshot](https://raw.githubusercontent.com/jackdoe/zearch/master/screenshot.gif)

# run

* indexing for the first time

```
2016/01/01 21:37:27 []string{"/SRC/glibc", "/SRC/go", "/SRC/jdk8", "/SRC/linux", "/SRC/musl", "/SRC/perl5"}
2016/01/01 21:37:27 starting indexer: 0/4
2016/01/01 21:37:27 starting indexer: 1/4
2016/01/01 21:37:27 starting indexer: 2/4
2016/01/01 21:37:27 starting indexer: 3/4
2016/01/01 21:38:38 indexing []string{"/SRC/glibc", "/SRC/go", "/SRC/jdk8", "/SRC/linux", "/SRC/musl", "/SRC/perl5"}: 71.630137s
2016/01/01 21:38:57 dumpToDisk /tmp/index.msgpack.lz4: 18.430408s
2016/01/01 21:38:57 listening to 8080
```

The index is msgpack encoded + lz4 compressed, and having the inverted + forward index of those source trees is about 80mb

```
$ du -ah /tmp/index.msgpack.lz4
71M     /tmp/index.msgpack.lz4
```

The whole index is kept in ram, above takes about 700mb for `66023 files` and `2013763 tokens` while indexing linux kernel, glibc, perl5, and jdk8

* if the index is ready

```
$ go run *.go  # without any arguments
2016/01/01 21:42:19 load /tmp/index.msgpack.lz4: 11.748887s
2016/01/01 21:42:19 listening to 8080
```

# search

 * just open http://localhost:8080

 * basenames can be searched with left edge ngrams so, `atomic.go` can be found with `a,at,ato,atom,atomic`, and the weight is increasing as they go closer to the full word

# search json

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

