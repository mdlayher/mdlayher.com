package web

import (
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

// A handler is a http.Handler that serves Content using a template.
type handler struct {
	c Content
}

// NewHandler creates a http.Handler that serves Content using a template.
func NewHandler(c Content) http.Handler {
	h := &handler{
		c: c,
	}

	mux := http.NewServeMux()
	mux.Handle("/", h)

	return mux
}

// ServeHTTP implements http.Handler.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// HSTS support: https://hstspreload.org/.
	w.Header().Set("Strict-Transport-Security", HSTSHeader(time.Now()))

	if err := tmpl.Execute(w, h.c); err != nil {
		log.Printf("failed to execute template: %v", err)
	}
}

// Content is the top-level object for the HTML template.
type Content struct {
	Domain  string
	Name    string
	Tagline string
	Links   []Link
}

// A Link is a hyperlink and a display title for that link.
type Link struct {
	Title string
	Link  string
}

// tmpl is the HTML template served to users of the site.
var tmpl = template.Must(template.New("html").Parse(strings.TrimSpace(`
<!DOCTYPE html>
<html>
<head>
	<title>{{.Name}}</title>
	<meta charset='utf-8' />
	<meta name="description" content="{{.Name}} - {{.Domain}}" />
</head>
<body>
	<h1>{{.Name}}</h1>
	<p>{{.Tagline}}</p>
	<ul>
		{{range .Links}}<li><a href="{{.Link}}">{{.Title}}</a></li>
		{{end}}
	</ul>
</body>
</html>
`)))
