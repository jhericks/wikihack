package main

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"fmt"
	"github.com/aymerick/raymond"
	"github.com/russross/blackfriday"
	"strings"
	"os"
	"log"
	"encoding/json"
)

var validPath = regexp.MustCompile("^/(view|edit|save|delete|admin)/([a-zA-Z0-9_]*)$")

type Account struct {
	Username string `json:"username"`
	Email string `json:"email"`
	GivenName string `json:"givenName"`
	MiddleName string `json:"middleName"`
	Surname string `json:"surname"`
	FullName string `json:"fullName"`
	Groups []string `json:"groups"`
}

func getIdentity(r *http.Request) *Account {
	accountHeader := r.Header.Get("X-Forwarded-Account")
	log.Println("Account Header: " + accountHeader)

	if len(accountHeader) == 0 {
		log.Println("Returning no account")
		return nil
	} else {
		var account *Account
		dec := json.NewDecoder(strings.NewReader(accountHeader))
		dec.Decode(&account)
		log.Printf("Returning Account %+v", account)
		return account
	}
}

type Page struct {
	Title string
	Body  []byte
	Account *Account
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

func (p *Page) IsFrontPage() bool {
	isFrontPage := p.Title == "FrontPage"
	return isFrontPage
}

func (p *Page) OtherPages() []*Page {
	fileInfoList, _ := ioutil.ReadDir("./data")
	pageList := []*Page{}
	for _, fileInfo := range fileInfoList {
		pageList = append(pageList, &Page{Title: strings.TrimSuffix(fileInfo.Name(), ".txt")})
	}
	return pageList
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

func makeUntitledHandler(fn func(http.ResponseWriter, *http.Request, *Account)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account := getIdentity(r)
		fn(w, r, account)
	}
}
func makeTitledHandler(fn func(http.ResponseWriter, *http.Request, *Account, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("makeHandler handling " + r.URL.Path)
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			fmt.Println("Write status not found header in makeHandler")
			w.WriteHeader(http.StatusNotFound)
			http.NotFound(w, r)
			return
		}
		fn(w, r, getIdentity(r), m[2])
	}

}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	title := "FrontPage"
	http.Redirect(w, r, "/view/" + title, http.StatusFound)
}

func viewHandler(w http.ResponseWriter, r *http.Request, account *Account, title string) {
	p, err := loadUserPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/" + title, http.StatusFound)
		return
	}
	p.Account = account
	renderTemplate(w, "view.html", p)
}

func adminHandler(w http.ResponseWriter, r *http.Request, account *Account) {
	title := "Admin"
	renderTemplate(w, "admin.html", &Page{Title: title, Account: account})
}

func editHandler(w http.ResponseWriter, r *http.Request, account *Account, title string) {
	p, err := loadUserPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	p.Account = account
	renderTemplate(w, "edit.html", p)
}

func createHandler(w http.ResponseWriter, r *http.Request, account *Account) {
	renderTemplate(w, "create.html", &Page{Title: "Create New Page", Account: account})
}

func saveHandler(w http.ResponseWriter, r *http.Request, account *Account, title string) {
	if len(title) == 0 {
		title = r.FormValue("title")
	}
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/" + title, http.StatusFound)
}

func deleteHandler(w http.ResponseWriter, r *http.Request, account *Account, title string) {
	if title == "FrontPage" {
		http.Error(w, "You cannot delete the front page", http.StatusBadRequest)
	}
	err := os.Remove("./data/" + title + ".txt")
	if err != nil {
		log.Fatal(err)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, templateName string, p *Page) {
	if p.Account == nil {
		log.Println("No account")
	} else {
		log.Println("Account for " + p.Account.FullName)
	}
	template, _ := raymond.ParseFile("templates/" + templateName)
	template.RegisterPartialFile("templates/layout_top.mustache", "layout_top")
	template.RegisterPartialFile("templates/layout_bottom.mustache", "layout_bottom")
	output, _ := template.Exec(&p)
	fmt.Fprint(w, output)
}

func main() {
	//fs := justFilesFilesystem{http.Dir("resources/")}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./public"))))
	http.HandleFunc("/admin", makeUntitledHandler(adminHandler))
	http.HandleFunc("/create/", makeUntitledHandler(createHandler))
	http.HandleFunc("/view/", makeTitledHandler(viewHandler))
	http.HandleFunc("/edit/", makeTitledHandler(editHandler))
	http.HandleFunc("/save/", makeTitledHandler(saveHandler))
	http.HandleFunc("/delete/", makeTitledHandler(deleteHandler))
	http.HandleFunc("/", rootHandler)
	http.ListenAndServe(":8080", nil)
}
