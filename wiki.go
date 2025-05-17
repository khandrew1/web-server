package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
)

// A Page represents a wiki page with a title and body.
// The body element is a byte slice instead of a string as this is type
// expeceted by the io libraries we're using
type Page struct {
	Title string
	Body  []byte
}

// Will panic if the regex fails to compile
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

// cache all our templates on first run, allowing all our templates to exist in a simple *Template
// template.Must will panic when a non-nil error value is passed to it
// Panicing is appropiate as if we can't load any templates, we shouldn't even run the server
var templates = template.Must(template.ParseFiles("tmpl/edit.html", "tmpl/view.html"))

// This function allows us to save our pages to disk, allowing for persistence storage
// This is a method named save that takes as its reciever p, a pointer to Page.
// Takes no parameters and returns an error type
func (p *Page) save() error {
	filename := p.Title + ".txt"
	return os.WriteFile("data/"+filename, p.Body, 0600)
}

// This function loadPage constructs our filename
// Reads that file from disk and returns a pointer to a Page struct
func loadPage(title string) (*Page, error) {
	filename := "data/" + title + ".txt"
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

// This renderTemplate function allows us to more easily write and execute our HTML files
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// function literal and closure that extracts the title from the URL and validates the path before passing it to a handler
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

// A function to actually server our pages to the browser
// The title of the page is extracted from the URL, minus the "/view/" prefix
func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

// This function handles our /edit/* path
// It returns a form that allows the user to
// edit the body of a function and then submit it to our save handler.
func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

// When the save button is hit on edit, it sends its form data to this handler
// This handler then extracts the body from the form and recreates the page
// It is then saved and redirected to the view page
// /save is used more as an API endpoint than a page
func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

// Handles our http requests and then listens and serves on port 8080
func main() {
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
