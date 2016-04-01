downloads
=========

`downloads` is an automatic download page generator. It takes a config file that describes the repositories to check, fetches release information for each of them, and generates a nice download page.

Running for development
-----------------------

- `go run *.go`
- windows users: `go build` then run downloads.exe
- Open http://localhost:8891/ in your browser
- Tweak the `index.html` and reload (reduce `TemplateCacheTime` in the config when developing).

Configuration
-------------

The following fields are used on the config.json:

- `ListenAddr`: The address on which to listen for HTTP requests.
- `RepoCacheTime`: How long to cache Github repo data (in nanoseconds).
- `TemplateCacheTime`: How long to cache evaluation of the `index.go.html` template (in nanoseconds).
- `Repos`: A list of repositories to present download data for. Each contains the following fields:
  - `FriendlyName`: What to call the project on the download page.
  - `GithubName`: What it's actually called on github (`owner/repo`).
  - `NameExpr`: What to call the download assets, either as a static `"string"` (used verbatim) or as a `"/regexp/"` where the first grouped match is used.
  - `ArchExpr`: As `NameExpr`, to extract the asset architecture.
  - `OSExpr`: As `NameExpr`, to extract the asset operating system.
