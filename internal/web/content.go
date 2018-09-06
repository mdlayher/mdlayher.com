package web

import (
	"html/template"
	"strings"

	"github.com/mdlayher/mdlayher.com/internal/github"
	"github.com/mdlayher/mdlayher.com/internal/httptalks"
	"github.com/mdlayher/mdlayher.com/internal/medium"
)

// Content is the top-level object for the HTML template.
type Content struct {
	Static    StaticContent
	GitHub    GitHubContent
	Medium    MediumContent
	HTTPTalks HTTPTalksContent
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

// MediumContent contains dynamic content from Medium for the HTML template.
type MediumContent struct {
	Posts []*medium.Post
}

// HTTPTalksContent contains dynamic content from an HTTP URL about talks
// for the HTML template.
type HTTPTalksContent struct {
	Talks []*httptalks.Talk
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
	<meta charset="utf-8" />
	<meta name="description" content="{{.Static.Name}} - {{.Static.Domain}}" />
	<style>
		* {
			font-family: sans-serif;
		}
	</style>
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
	{{if .Medium.Posts}}
	<h2>Blog Posts</h2>
	<ul>
	{{range .Medium.Posts}}<li><a href="{{.Link}}">{{.Title}}</a></li>
	<ul><li>{{.Subtitle}}</li></ul>
	{{end}}
	</ul>
	{{end}}
	{{if .HTTPTalks.Talks}}
	<h2>Talks</h2>
	<ul>
	{{range .HTTPTalks.Talks}}<li>{{if .VideoLink}}<a href="{{.VideoLink}}">{{.Title}}</a>{{else}}{{.Title}}{{end}}</li>
		<ul>
		{{if .Description}}<li>{{.Description}}</li>{{end}}
		<li>{{if .AudioLink}} [<a href="{{.AudioLink}}">audio</a>]{{end}}{{if .BlogLink}} [<a href="{{.BlogLink}}">blog</a>]{{end}}{{if .SlidesLink}} [<a href="{{.SlidesLink}}">slides</a>]{{end}}</li>
		</ul>
	{{end}}
	</ul>
	{{end}}
</body>
</html>
`)))

//
