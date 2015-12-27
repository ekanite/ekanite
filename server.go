package ekanite

import (
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/justinas/nosurf"
)

type Searcher interface {
	Search(query string) (<-chan string, error)
}

// Server serves query client connections.
type Server struct {
	iface    string
	Searcher Searcher
	addr     net.Addr
	template *template.Template

	Logger *log.Logger
}

// NewServer returns a new Server instance.
func NewServer(iface string, searcher Searcher) *Server {
	return &Server{
		iface:    iface,
		Searcher: searcher,
		Logger:   log.New(os.Stderr, "[server] ", log.LstdFlags),
	}
}

// Start instructs the Server to bind to the interface and accept connections.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.iface)
	if err != nil {
		return err
	}

	s.template, err = template.New("ServerTemplate").Parse(templateSource)
	if err != nil {
		ln.Close()
		return err
	}

	csrfHandler := nosurf.New(s)
	csrfHandler.SetBaseCookie(http.Cookie{HttpOnly: true})

	go http.Serve(ln, csrfHandler)

	s.addr = ln.Addr()
	return nil
}

// Addr returns the address to which the Server is bound.
func (s *Server) Addr() net.Addr {
	return s.addr
}

// ServeHTTP implements a http.Handler, serving the query interface for Ekanite
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	dontCache(w, r)

	if r.Method == "GET" || r.Method == "HEAD" {
		//HEAD is conveniently supported by net/http without further action
		serveIndex(s, w, r)

	} else if r.Method == "POST" {

		err := r.ParseForm()
		if err != nil {
			s.Logger.Printf("Error parsing form '%s'", err)
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		if len(r.FormValue("query")) == 0 {
			serveIndex(s, w, r)
			return
		}

		userQuery := r.FormValue("query")

		s.Logger.Printf("executing query '%s'", userQuery)

		resultSet, err := s.Searcher.Search(userQuery)
		var resultSlice []string

		if err != nil {
			s.Logger.Printf("Error executing query: '%s'", err)
			http.Error(w, "Error executing query: "+err.Error(), http.StatusInternalServerError)
			return
		} else {
			for s := range resultSet {
				resultSlice = append(resultSlice, s)
			}
		}

		data := struct {
			Token         string
			Title         string
			Headline      string
			ReturnResults bool
			LogMessages   []string
		}{
			nosurf.Token(r),
			"Ekanite query interface",
			"Ekanite - Listing " + strconv.Itoa(len(resultSlice)) + " results for '" + userQuery + "'",
			true,
			resultSlice,
		}

		if err := s.template.Execute(w, data); err != nil {
			s.Logger.Print("Error executing template: ", err)
		}

	} else {
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
}

// serveIndex serves the plain index for the GET request and POST failovers
func serveIndex(s *Server, w http.ResponseWriter, r *http.Request) {
	data := struct {
		Token         string
		Title         string
		Headline      string
		ReturnResults bool
		LogMessages   []string
	}{
		nosurf.Token(r),
		"Ekanite query interface",
		"Ekanite query interface",
		false,
		[]string{},
	}

	if err := s.template.Execute(w, data); err != nil {
		s.Logger.Print("Error executing template: ", err)
	}

}

// dontCache sets necessary headers to avoid client and intermediate caching of response
func dontCache(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Expires", time.Unix(0, 0).Format(time.RFC1123))
	w.Header().Set("Last-Modified", time.Now().Format(time.RFC1123))
	w.Header().Set("Cache-Control", "private, no-store, max-age=0, no-cache, must-revalidate, post-check=0, pre-check=0")

	return
}

const templateSource string = `
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8" />
<title>{{ $.Title }}</title>
<style type="text/css"> 

body, h3 {
margin: 50px;
font-family: sans-serif;
font-size: 13px;
}

h3 {
font-size: 15px;
}

.button {
background: #3498db;
background-image: linear-gradient(to bottom, #3498db, #2980b9);
border-radius: 4px;
font-family: sans-serif;
color: #ffffff;
font-size: 15px;
padding: 10px 20px 10px 20px;
margin-bottom: 20px;
text-decoration: none;
}

hr {
	margin-bottom: 10px;
	margin-top: 10px;
}

.button:hover {
background: #3cb0fd;
background-image: linear-gradient(to bottom, #3cb0fd, #3498db);
text-decoration: none;
}

textarea {
margin: 20px 20px 20px 0;
}

</style>
</head>
<body>

	<h2>{{ $.Headline }}</h2>
	<div id="help">Query language reference: <a href="http://www.blevesearch.com/docs/Query-String-Query/">Bleve Query Strings</a></div>
	<form action="/" method="POST">
    <textarea name="query" cols="100" rows="2"></textarea>
    <br>
    <input name="submit" type="submit" class="button" value="Query">
    <input name="csrf_token" type="hidden" value="{{ $.Token }}">
	</form>
	
{{ if $.ReturnResults }}

	<hr>
	<ul>
	{{range $message := $.LogMessages }}
	<li>{{ $message }}</li>
	{{ end }}
	</ul>

{{ end }}


</body>
</html>
`
