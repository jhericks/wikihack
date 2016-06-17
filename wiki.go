package main

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"fmt"
	"github.com/aymerick/raymond"
	"github.com/russross/blackfriday"
	"strings"
)

var validPath = regexp.MustCompile("^/(edit|save|view|admin)/([a-zA-Z0-9]+)$")

type Page struct {
	Title string
	Body  []byte
}

func (p *Page) BodyStr() string {
	return string(p.Body)
}

func (p *Page) BodyHtml() string {
	return string(blackfriday.MarkdownCommon(p.Body))
}

func (p *Page) save() error {
	filename := "data/" + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func (p *Page) OtherPages() []string {
	fileInfoList, _ := ioutil.ReadDir("./data")
	fileList := []string{}
	for _,fileInfo := range fileInfoList {
		fileList = append(fileList, strings.TrimSuffix(fileInfo.Name(), ".txt"))
	}
	return fileList
}

func loadPage(title string, filename string) (*Page, error) {
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	} else {
		return &Page{Title: title, Body: body}, nil
	}
}

func loadUserPage(title string) (*Page, error) {
	filename := "data/" + title + ".txt"
	return loadPage(title, filename)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("makeHandler handling " + r.URL.Path)
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			fmt.Println("Write status not found header in makeHandler")
			w.WriteHeader(http.StatusNotFound)
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}

}


func rootHandler(w http.ResponseWriter, r *http.Request) {
	title := "FrontPage"
	http.Redirect(w, r, "/view/" + title, http.StatusFound)
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadUserPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/" + title, http.StatusFound)
		return
	}

	renderTemplate(w, "view.html", p)
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	title := "Admin"
	renderTemplate(w, "admin.html", &Page{Title: title})
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadUserPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit.html", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/" + title, http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, templateName string, p *Page) {
	template, _ := raymond.ParseFile("templates/" + templateName)
	template.RegisterPartialFile("templates/layout_top.mustache", "layout_top")
	template.RegisterPartialFile("templates/layout_bottom.mustache", "layout_bottom")
	output, _ := template.Exec(&p)
	fmt.Fprint(w, output)
}

func main() {
	//fs := justFilesFilesystem{http.Dir("resources/")}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./public"))))
	http.HandleFunc("/admin", adminHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/", rootHandler)
	http.ListenAndServe(":8080", nil)
}
