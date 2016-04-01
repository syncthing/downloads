package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"
)

type handler struct {
	cfg       config
	mu        sync.Mutex
	repoCache struct {
		repos   []repo
		updated time.Time
	}
	templateCache struct {
		data    []byte
		updated time.Time
	}
}

func (h *handler) handle(w http.ResponseWriter, req *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if we need to reload Github data.
	// FIXME: Should happen in the background as it can take quite a while.
	repoDataUpdated := false
	if time.Since(h.repoCache.updated) > h.cfg.RepoCacheTime {
		h.repoCache.repos = getAllRepositories(h.cfg)
		h.repoCache.updated = time.Now()
		repoDataUpdated = true
	}

	// Check if we need to re-evaluate the template.
	if repoDataUpdated || time.Since(h.templateCache.updated) > h.cfg.TemplateCacheTime {
		bs, err := generateOverview(h.repoCache.repos)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.templateCache.data = bs
		h.templateCache.updated = time.Now()
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(h.templateCache.data)
}

func getAllRepositories(cfg config) []repo {
	var rs []repo
	for _, rc := range cfg.Repos {
		r, err := getRepository(rc)
		if err != nil {
			log.Printf("Getting %s: %v", rc.GithubName, err)
			continue
		}
		rs = append(rs, r)
	}

	return rs
}

func getRepository(rc repoCfg) (repo, error) {
	gr, err := getRepo(rc.GithubName)
	if err != nil {
		return repo{}, err
	}

	rs, err := gr.getReleases(5) // At most five releases per repo
	if err != nil {
		return repo{}, err
	}

	r := repo{
		FriendlyName: rc.FriendlyName,
		GithubName:   gr.FullName,
		CssName:      strings.ToLower(strings.Replace(rc.FriendlyName, " ", "-", -1)),
		IsSyncthing:  strings.Compare(rc.FriendlyName, "Syncthing")==0,
		Description:  gr.Description,
		GithubURL:    gr.URL,
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
				log.Printf("%s %s: %v (skipping)", rc.GithubName, rel.Tag, err)
				continue
			} else {
				ast.Name = val
			}

			if val, err := extractExpr(rc.OSExpr, as.Name); err != nil {
				log.Printf("%s %s: %v (skipping)", rc.GithubName, rel.Tag, err)
				continue
			} else {
				ast.OS = val
			}

			if val, err := extractExpr(rc.ArchExpr, as.Name); err != nil {
				log.Printf("%s %s: %v (skipping)", rc.GithubName, rel.Tag, err)
				continue
			} else {
				ast.Arch = val
			}

			ver.Assets = append(ver.Assets, ast)
		}
		r.Versions = append(r.Versions, ver)
	}

	return r, nil
}

func generateOverview(repos []repo) ([]byte, error) {
	fm := template.FuncMap{
		"now": time.Now,
	}
	tmpl := template.Must(template.New("index.go.html").Funcs(fm).ParseGlob("*.go.html"))

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, repos); err != nil {
		log.Printf("Template execution: %s", err)
		return nil, err
	}

	return buf.Bytes(), nil
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
