package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed web/*
var webFiles embed.FS

func (h *Handler) serveWeb(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(request.URL.Path, "/")
	path = strings.TrimPrefix(path, "assets/")
	if path == "" {
		path = "index.html"
	}

	files, err := fs.Sub(webFiles, "web")
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "web files are unavailable")
		return
	}

	file, err := files.Open(path)
	if err != nil {
		if request.URL.Path != "/" {
			http.NotFound(writer, request)
			return
		}
		path = "index.html"
	}
	if file != nil {
		file.Close()
	}

	if path == "index.html" {
		index, err := fs.ReadFile(files, "index.html")
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "index page is unavailable")
			return
		}

		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = writer.Write(index)
		return
	}

	webRequest := request.Clone(request.Context())
	webRequest.URL.Path = "/" + path
	http.FileServer(http.FS(files)).ServeHTTP(writer, webRequest)
}
