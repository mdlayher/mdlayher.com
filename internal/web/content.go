package web

import (
	"html/template"
	"strings"

	"github.com/mdlayher/mdlayher.com/internal/github"
)

// Content is the top-level object for the HTML template.
type Content struct {
	Static StaticContent
	GitHub GitHubContent
}

// StaticContent contains statically defined content for the HTML template.
type StaticContent struct {
	Domain  string
	Name    string
	Tagline string
	Links   []Link
}

// GitHubContent contains dynamic content from GitHub for the HTML template.
type GitHubContent struct {
	Repositories []*github.Repository
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
	<title>{{.Static.Name}}</title>
	<meta charset='utf-8' />
	<meta name="description" content="{{.Static.Name}} - {{.Static.Domain}}" />
</head>
<body>
	<h1>{{.Static.Name}}</h1>
	<p>{{.Static.Tagline}}</p>
	<ul>
		{{range .Static.Links}}<li><a href="{{.Link}}">{{.Title}}</a></li>
		{{end}}
	</ul>
	{{if .GitHub.Repositories}}
	<h2>Open Source</h2>
		<ul>
		{{range .GitHub.Repositories}}<li><a href="{{.Link}}">{{.Name}}</a></li>
		<ul><li>{{.Description}}</li></ul>
		{{end}}
		</ul>
	{{end}}
</body>
</html>
`)))
