package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/codegangsta/martini"
	"github.com/martini-contrib/render"
)

func get_pkg_page(params martini.Params, req *http.Request, log *log.Logger, r render.Render) {
	if req.URL.Query().Get("go-get") == "1" {
		log.Printf("headers=%+v", req.Header)
	}

	r.HTML(200, "pkg", &PkgPage{params["owner"], params["repo"], params["version"], params["_1"]})
}

type PkgPage struct {
	Owner   string
	Repo    string
	Version string
	Rest    string
}

func (page *PkgPage) PkgPath() string {
	if page.Rest != "" {
		return fmt.Sprintf("%s/%s", page.BasePath(), page.Rest)
	}
	return page.BasePath()
}

func (page *PkgPage) SrcPath() string {
	if page.Rest != "" {
		fmt.Sprintf("%s/%s/tree/%s/%s", page.Owner, page.Repo, page.Version, page.Rest)
	}
	return fmt.Sprintf("%s/%s/tree/%s", page.Owner, page.Repo, page.Version)
}

func (page *PkgPage) BasePath() string {
	return fmt.Sprintf("gopkg.cc/%s/%s/%s", page.Owner, page.Repo, page.Version)
}
