package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// The types in this file are the ones returned by the Github API. We
// convert these to internal types that better fit our use case - see
// assets.go.

// A Github repository
type githubRepo struct {
	Name        string
	FullName    string `json:"full_name"`
	Description string
	URL         string `json:"html_url"`
	ReleasesURL string `json:"releases_url"`
}

// getRepo returns a Github repository, given it's name like
// "syncthing/syncthing".
func getRepo(name string) (githubRepo, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s", name))
	if err != nil {
		return githubRepo{}, err
	}
	defer resp.Body.Close()

	var r githubRepo
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return githubRepo{}, err
	}

	return r, nil
}

// getReleases returns the list of releases for a given repo, up to n of
// them.
func (r githubRepo) getReleases(n int) ([]githubRelease, error) {
	url := strings.Replace(r.ReleasesURL, "{/id}", "", 1)
	resp, err := http.Get(fmt.Sprintf("%s?per_page=%d", url, n))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rs []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rs); err != nil {
		return nil, err
	}

	return rs, nil
}

// A Github release (i.e. a specific tag on a repository).
type githubRelease struct {
	Name   string
	Tag    string `json:"tag_name"`
	Assets []githubAsset
}

// A Github release asset (i.e. a specific file within a release).
type githubAsset struct {
	URL   string `json:"browser_download_url"`
	Name  string
	Label string
}
