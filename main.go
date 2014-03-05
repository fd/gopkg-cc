package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/codegangsta/martini"
	"github.com/martini-contrib/render"
)

func main() {
	m := martini.Classic()

	m.Use(render.Renderer())

	m.Get("/:owner/:repo/:version/info/refs", proxy_refs)
	m.Post("/:owner/:repo/:version/git-upload-pack", proxy_upload_pack)

	m.Get("/:owner/:repo/:version/**", get_pkg_page)
	m.Get("/:owner/:repo/:version", get_pkg_page)

	m.Run()
}

func proxy_upload_pack(req *http.Request, rw http.ResponseWriter, params martini.Params) {
	url := fmt.Sprintf("http://github.com/%s/%s.git/git-upload-pack", params["owner"], params["repo"])

	resp, err := http.Post(url, "application/x-git-upload-pack-request", req.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		rw.Header()[k] = v
	}

	rw.WriteHeader(resp.StatusCode)

	_, err = io.Copy(rw, resp.Body)
	if err != nil {
		panic(err)
	}
}

func proxy_refs(req *http.Request, rw http.ResponseWriter, params martini.Params) {
	url := fmt.Sprintf("http://github.com/%s/%s.git/info/refs?service=git-upload-pack", params["owner"], params["repo"])

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		for k, v := range resp.Header {
			rw.Header()[k] = v
		}
		rw.WriteHeader(resp.StatusCode)
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			panic(err)
		}
		return
	}

	params["version"] = strings.TrimSuffix(params["version"], ".git")
	r := GitChunkReader(resp.Body)

	lines1, err := read_refs(r, "-----")
	if err != nil {
		panic(err)
	}

	lines2, err := read_refs(r, params["version"])
	if err != nil {
		panic(err)
	}
	lines2, found := rewrite_ref_lines(lines2, params["version"])

	if !found {
		http.NotFound(rw, req)
		return
	}

	for k, v := range resp.Header {
		rw.Header()[k] = v
	}
	rw.WriteHeader(resp.StatusCode)
	_, err = format_lines(rw, lines1)
	if err != nil {
		panic(err)
	}
	_, err = format_lines(rw, lines2)
	if err != nil {
		panic(err)
	}
}

func format_lines(w io.Writer, lines []string) (int, error) {
	n := 0
	for _, line := range lines {
		i, err := fmt.Fprintf(w, "%04x%s\n", len(line)+5, line)
		n += i
		if err != nil {
			return n, err
		}
	}
	i, err := fmt.Fprint(w, "0000")
	n += i
	return n, err
}

func rewrite_ref_lines(lines []string, ref string) ([]string, bool) {
	var (
		branch_ref = fmt.Sprintf("refs/heads/%s", ref)
		tag_ref    = fmt.Sprintf("refs/tags/%s", ref)
		commit_sha string
	)

	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 1 {
			continue
		}

		if parts[1] == branch_ref || parts[1] == tag_ref {
			commit_sha = parts[0]
		}
	}

	if commit_sha == "" {
		return nil, false
	}

	for i, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 1 {
			continue
		}

		if strings.HasPrefix(parts[1], "HEAD") {
			lines[i] = fmt.Sprintf("%s %s", commit_sha, parts[1])
		}

		if parts[1] == branch_ref || parts[1] == tag_ref {
			lines[i] = fmt.Sprintf("%s refs/heads/master", commit_sha)
		}
	}

	return lines, true
}

func read_refs(r io.Reader, ref string) ([]string, error) {
	var (
		lines      []string
		branch_ref = fmt.Sprintf("refs/heads/%s", ref)
		tag_ref    = fmt.Sprintf("refs/tags/%s", ref)
	)

	for {
		line, err := read_line(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(line, "#") {
			lines = append(lines, line)
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 1 {
			lines = append(lines, line)
			continue
		}

		if strings.HasPrefix(parts[1], "HEAD") {
			lines = append(lines, line)
			continue
		}

		if parts[1] == branch_ref || parts[1] == tag_ref {
			lines = append(lines, line)
		}
	}

	return lines, nil
}

func read_line(r io.Reader) (string, error) {
	var (
		size int
		line []byte
	)

	{
		var (
			size_buf [4]byte
		)

		_, err := io.ReadFull(r, size_buf[:])
		if err != nil {
			return "", err
		}

		i64, err := strconv.ParseInt(string(size_buf[:]), 16, 32)
		if err != nil {
			return "", err
		}

		if i64 == 0 {
			return "", io.EOF
		}

		size = int(i64) - 4
	}

	line = make([]byte, size)
	_, err := io.ReadFull(r, line)
	if err != nil {
		return "", err
	}

	return string(line[:size-1]), nil
}
