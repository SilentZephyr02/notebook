package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
)

/*
type Members struct {
	ID       int
	Password string
}

type Presets struct {
	ID          int
	OwnerID     int
	MemberID    int
	Permissions int
}

type MetaNote struct {
	NoteID      int
	MemberID    int
	Permissions int
}

type Note struct {
	ID   int
	Note string
}
*/

//localhost:8080

var tpl *template.Template

func init() {
	db, err := sql.Open("postgres", "user=postgres dbname=testDB sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		panic(err)
	}
	fmt.Println("You connected to your database.")

	tpl = template.Must(template.ParseGlob("templates/*"))
}

func main() {
	http.HandleFunc("/", index)
	http.HandleFunc("/about", about)
	http.ListenAndServe(":8080", nil)
}

func index(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "index.gohtml", "ACME INC")
}

func about(w http.ResponseWriter, r *http.Request) {
	type customData struct {
		Title   string
		Members []string
	}

	cd := customData{
		Title: "THE TEAM", Members: []string{"Money", "Wee", "Kay"},
	}

	tpl.ExecuteTemplate(w, "about.gohtml", cd)
}
