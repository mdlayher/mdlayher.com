package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mdlayher/mdlayher.com/internal/github"
	"github.com/mdlayher/mdlayher.com/internal/medium"
)

// A handler is a http.Handler that serves content using a template.
type handler struct {
	static   StaticContent
	redirect http.Handler
	ghc      github.Client
	mc       medium.Client
}

// NewHandler creates a http.Handler that serves content using a template.
// Additional dynamic content can be added by providing non-nil clients for
// various services.
func NewHandler(static StaticContent, ghc github.Client, mc medium.Client) http.Handler {
	h := &handler{
		static:   static,
		redirect: NewRedirectHandler(static.Domain),
		ghc:      ghc,
		mc:       mc,
	}

	mux := http.NewServeMux()
	mux.Handle("/", h)

	return mux
}

// ServeHTTP implements http.Handler.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Redirect any TLS subdomain requests to the base domain name.
	if r.TLS != nil && strings.Count(r.TLS.ServerName, ".") > 1 {
		h.redirect.ServeHTTP(w, r)
		return
	}

	// HSTS support: https://hstspreload.org/.
	w.Header().Set("Strict-Transport-Security", HSTSHeader(time.Now()))

	// Build content for display.
	content := Content{
		Static: h.static,
	}

	// If available, add GitHub content.
	if h.ghc != nil {
		repos, err := h.ghc.ListRepositories(context.Background())
		if err != nil {
			httpError(w, "failed to retrieve github repositories: %v", err)
			return
		}

		content.GitHub = GitHubContent{
			Repositories: repos,
		}
	}

	// If available, add Medium content.
	if h.mc != nil {
		posts, err := h.mc.ListPosts()
		if err != nil {
			httpError(w, "failed to retrieve medium posts: %v", err)
			return
		}

		content.Medium = MediumContent{
			Posts: posts,
		}
	}

	if err := tmpl.Execute(w, content); err != nil {
		httpError(w, "failed to execute template: %v", err)
		return
	}
}

// NewRedirectHandler creates a http.Handler that redirects clients to the
// specified domain using TLS and no subdomain.
func NewRedirectHandler(domain string) http.Handler {
	return http.RedirectHandler(fmt.Sprintf("https://%s", domain), http.StatusMovedPermanently)
}

// httpError returns a generic HTTP 500 to a client and logs an informative
// message to the logger.
func httpError(w http.ResponseWriter, format string, a ...interface{}) {
	log.Printf(format, a...)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
