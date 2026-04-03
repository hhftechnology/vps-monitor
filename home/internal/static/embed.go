package static

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed dist/*
var distFS embed.FS

// GetFileSystem returns the embedded filesystem with the dist directory as root
func GetFileSystem() (http.FileSystem, error) {
	// Strip the "dist" prefix so files are served from root
	fsys, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(fsys), nil
}

// SPAHandler wraps a http.FileServer to serve index.html for SPA routes
type SPAHandler struct {
	staticFS   http.FileSystem
	fileServer http.Handler
}

func NewSPAHandler(staticFS http.FileSystem) *SPAHandler {
	return &SPAHandler{
		staticFS:   staticFS,
		fileServer: http.FileServer(staticFS),
	}
}

func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}
	upath = path.Clean(upath)

	f, err := h.staticFS.Open(upath)
	if err == nil {
		f.Close()
		h.fileServer.ServeHTTP(w, r)
		return
	}

	indexPath := path.Join(upath, "index.html")
	f, err = h.staticFS.Open(indexPath)
	if err == nil {
		f.Close()
		h.fileServer.ServeHTTP(w, r)
		return
	}

	// Fall back to index.html for SPA routing (client-side routing)
	indexFile, err := h.staticFS.Open("/index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer indexFile.Close()

	http.ServeContent(w, r, "index.html", time.Time{}, indexFile.(io.ReadSeeker))
}
