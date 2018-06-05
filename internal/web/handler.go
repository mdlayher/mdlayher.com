package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mdlayher/mdlayher.com/internal/github"
	"github.com/mdlayher/mdlayher.com/internal/httptalks"
	"github.com/mdlayher/mdlayher.com/internal/medium"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// A handler is a http.Handler that serves content using a template.
type handler struct {
	static   StaticContent
	redirect http.Handler
	ghc      github.Client
	mc       medium.Client
	htc      httptalks.Client

	requestDurationSeconds *prometheus.HistogramVec
}

// NewHandler creates a http.Handler that serves content using a template.
// Additional dynamic content can be added by providing non-nil clients for
// various services.
func NewHandler(static StaticContent, ghc github.Client, mc medium.Client, htc httptalks.Client) http.Handler {
	const namespace = "mdlayher"

	h := &handler{
		static:   static,
		redirect: NewRedirectHandler(static.Domain),
		ghc:      ghc,
		mc:       mc,
		htc:      htc,

		requestDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "request_duration_seconds",
			Help:      "Duration of requests to external services.",
			Buckets:   prometheus.ExponentialBuckets(0.1, 2, 7),
		}, []string{"target"}),
	}

	prometheus.MustRegister(h.requestDurationSeconds)

	// Set up application routes and metrics.
	mux := http.NewServeMux()
	mux.Handle("/", prometheus.InstrumentHandler("web", h))
	mux.Handle("/metrics", promhttp.Handler())

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

	// Dispatch a group of futures to fetch external data, to be evaluated
	// once ready to populate the Content structure.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoFn := h.fetchGitHub(ctx)
	postFn := h.fetchMedium(ctx)
	talkFn := h.fetchTalks(ctx)

	repos, err := repoFn()
	if err != nil {
		httpError(w, "failed to fetch github repositories: %v", err)
		return
	}

	posts, err := postFn()
	if err != nil {
		httpError(w, "failed to fetch medium posts: %v", err)
		return
	}

	talks, err := talkFn()
	if err != nil {
		httpError(w, "failed to fetch talks: %v", err)
		return
	}

	// Build content for display.
	content := Content{
		Static: h.static,
		GitHub: GitHubContent{
			Repositories: repos,
		},
		Medium: MediumContent{
			Posts: posts,
		},
		HTTPTalks: HTTPTalksContent{
			Talks: talks,
		},
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
