package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

var ROOT *string
var INDEX *string
var URL *string

type repository struct {
	Url string
	Dir string
}

func (r *repository) exec(name string, arg ...string) []byte {
	log.Printf("%#v: exec %s %#v", r, name, arg)
	out, err := exec.Command(name, arg...).Output()

	if err != nil {
		log.Fatalf("%#v: failed return code: %+v, output: %s", r, err, string(out))
	}
	return out
}

func (r *repository) path() string {
	return fmt.Sprintf("%s/%s", *ROOT, r.Dir)
}

func (r *repository) exists() bool {
	if _, err := os.Stat(r.path()); err == nil {
		return true
	}
	return false
}

func (r *repository) clone_if_not_exists() {
	if !r.exists() {
		os.MkdirAll(r.path(), 0755)
		out := r.exec("git", "clone", r.Url, r.path())
		log.Printf(string(out))
	}
}

func (r *repository) pull() {
	os.MkdirAll(r.path(), 0755)
	out := r.exec("git", fmt.Sprintf("--work-tree=%s", r.path()), fmt.Sprintf("--git-dir=%s/.git", r.path()), "pull")
	log.Printf(string(out))
}

func main() {
	current := 0
	old_body := []byte{}

	ROOT = flag.String("dir-to-index", "/SRC", "directory to index")
	INDEX := flag.String("dir-to-store", "/tmp/zearch", "directory to store the index")
	URL := flag.String("url", "", "config url")
	flag.Parse()
	if len(*URL) == 0 {
		log.Fatalf("need -url argument for config.json (see https://raw.githubusercontent.com/jackdoe/zearch/master/zearch.io/config.json)")
	}

	name_for_iteration := func(i int) string {
		return fmt.Sprintf("%s.%d", *INDEX, i)
	}
	update := func() {
		data := []repository{}
		r, err := http.Get(*URL)
		if err != nil {
			log.Print(err)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			log.Print(err)
			return
		}
		if err := json.Unmarshal(body, &data); err != nil {
			log.Print(err)
			return
		}

		if bytes.Compare(body, old_body) > 0 {
			for _, r := range data {
				r.clone_if_not_exists()
				r.pull()
			}

			name := name_for_iteration(current)
			remove(name_for_iteration(current - 2))
			remove(name)
			exec_dont_care("zearch", "-dir-to-index", *ROOT, "-dir-to-store", name)
			tmp := fmt.Sprintf("%s.lnk", name)
			exec_dont_care("ln", "-vs", name, tmp)
			exec_dont_care("mv", "-v", tmp, *INDEX)
			exec_dont_care("pkill", "--signal", "1", "zearch")
			current++

			old_body = body
		}
	}

	for {
		update()
		time.Sleep(10000 * time.Millisecond)
	}
}

func exec_dont_care(name string, arg ...string) []byte {
	out, err := exec.Command(name, arg...).Output()
	log.Printf("%s %#v = %s, [ %+v] ", name, arg, out, err)
	return out
}

func remove(name string) {
	exec_dont_care("rm", "-rvf", name)
}