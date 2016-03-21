package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type githubRepo struct {
	Name        string
	FullName    string `json:"full_name"`
	Description string
	URL         string `json:"html_url"`
	ReleasesURL string `json:"releases_url"`
}

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

type githubRelease struct {
	Name   string
	Tag    string `json:"tag_name"`
	Assets []githubAsset
}

type githubAsset struct {
	URL   string `json:"browser_download_url"`
	Name  string
	Label string
}

func (r githubRepo) getReleases() ([]githubRelease, error) {
	url := strings.Replace(r.ReleasesURL, "{/id}", "", 1)
	resp, err := http.Get(url + "?per_page=5")
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
