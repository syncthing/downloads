package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type config struct {
	ListenAddr string
	CacheTime  time.Duration
	Repos      []repoCfg
}

type repoCfg struct {
	FriendlyName string
	GithubName   string
	NameExpr     string
	ArchExpr     string
	OSExpr       string
}

var cfg config

func main() {
	cfgFile := flag.String("cfg", "config.json", "Name of configuration file")
	flag.Parse()

	fd, err := os.Open(*cfgFile)
	if err != nil {
		log.Fatal("Reading config:", err)
	}

	if err := json.NewDecoder(fd).Decode(&cfg); err != nil {
		log.Fatal("Reading config:", err)
	}

	fd.Close()

	http.HandleFunc("/", handle)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, nil))
}

var cache = struct {
	data    []byte
	updated time.Time
	sync.Mutex
}{}

func handle(w http.ResponseWriter, req *http.Request) {
	cache.Lock()
	defer cache.Unlock()

	if time.Since(cache.updated) > cfg.CacheTime {
		cache.data = generateOverview()
		cache.updated = time.Now()
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(cache.data)
}

func getDownloads(rc repoCfg) (downloads, error) {
	gr, err := getRepo(rc.GithubName)
	if err != nil {
		return downloads{}, err
	}

	rs, err := gr.getReleases()
	if err != nil {
		return downloads{}, err
	}

	ds := downloads{
		Repo: repo{
			FriendlyName: rc.FriendlyName,
			GithubName:   gr.FullName,
			Description:  gr.Description,
			GithubURL:    gr.URL,
		},
	}

	for _, rel := range rs {
		ver := version{
			Version: rel.Tag,
		}
		for _, as := range rel.Assets {
			ast := asset{
				URL: as.URL,
			}

			if val, err := extractExpr(rc.NameExpr, as.Name); err != nil {
				log.Println("Extracting NameExpr:", err)
				continue
			} else {
				ast.Name = val
			}

			if val, err := extractExpr(rc.OSExpr, as.Name); err != nil {
				log.Println("Extracting OSExpr:", err)
				continue
			} else {
				ast.OS = val
			}

			if val, err := extractExpr(rc.ArchExpr, as.Name); err != nil {
				log.Println("Extracting ArchExpr:", err)
				continue
			} else {
				ast.Arch = val
			}

			fmt.Printf("%#v\n", ast)

			ver.Assets = append(ver.Assets, ast)
		}
		ds.Versions = append(ds.Versions, ver)
	}

	return ds, nil
}

func getAllDownloads() []downloads {
	var ds []downloads
	for _, rc := range cfg.Repos {
		d, err := getDownloads(rc)
		if err != nil {
			log.Println("Getting", rc.GithubName, ":", err)
			continue
		}
		ds = append(ds, d)
	}

	return ds
}

func generateOverview() []byte {
	fm := template.FuncMap{
		"now": time.Now,
	}
	tmpl := template.Must(template.New("index.go.html").Funcs(fm).ParseGlob("*.go.html"))

	var dls = getAllDownloads()
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, dls); err != nil {
		log.Fatalf("template execution: %s", err)
	}

	return buf.Bytes()
}

func extractExpr(expr, val string) (string, error) {
	if !strings.HasPrefix(expr, "/") {
		return expr, nil
	}

	re, err := regexp.Compile(expr[1 : len(expr)-1])
	if err != nil {
		return "", err
	}

	ms := re.FindStringSubmatch(val)
	if len(ms) != 2 {
		return "", fmt.Errorf("%s doesn't match %q", expr, val)
	}

	return ms[1], nil
}
