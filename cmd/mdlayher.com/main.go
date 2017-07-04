// Command mdlayher.com serves Matt Layher's personal website.
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/acme/autocert"
)

func main() {
	// Information is hard-coded for simplicity of deployment, but this could
	// be easily changed in the future.
	c := content{
		Domain:  "mdlayher.com",
		Name:    "Matt Layher",
		Tagline: "Software Engineer. Go, Linux, and open source software enthusiast. On and ever upward.",
		Links: []link{
			{
				Title: "GitHub",
				Link:  "https://github.com/mdlayher",
			},
			{
				Title: "Medium (blog)",
				Link:  "https://medium.com/@mdlayher",
			},
			{
				Title: "Twitter",
				Link:  "https://twitter.com/mdlayher",
			},
		},
	}

	// Always redirect HTTP to HTTPS.
	go func() {
		log.Println("starting HTTP redirect server")

		h := http.RedirectHandler(fmt.Sprintf("https://%s", c.Domain), http.StatusMovedPermanently)
		if err := http.ListenAndServe(":80", h); err != nil {
			log.Fatalf("failed to serve HTTP: %v", err)
		}
	}()

	log.Printf("starting HTTPS server for domain %q", c.Domain)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// HSTS support: https://hstspreload.org/.
		w.Header().Set("Strict-Transport-Security", "max-age=300; includeSubDomains")

		if err := tmpl.Execute(w, c); err != nil {
			log.Printf("failed to execute template: %v", err)
		}
	})

	domains := []string{
		c.Domain,
		// Also include www subdomain.
		"www." + c.Domain,
	}

	if err := http.Serve(autocert.NewListener(domains...), mux); err != nil {
		log.Fatalf("failed to serve HTTPS: %v", err)
	}
}

// content is the top-level object for the HTML template.
type content struct {
	Domain  string
	Name    string
	Tagline string
	Links   []link
}

// A link is a hyperlink and a display title for that link.
type link struct {
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
