package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"

	_ "github.com/lib/pq"
)

type Members struct {
	ID       int
	Password string
}

/*
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

var db *sql.DB
var tpl *template.Template

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "password"
	dbname   = "notebook"
)

func init() {
	var err error
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err = sql.Open("postgres", psqlInfo)

	if err != nil {
		panic(err)
	}

	fmt.Println("You connected to the database.")
	_, err = db.Exec("CREATE TABLE  IF NOT EXISTS members (ID SERIAL PRIMARY KEY,Password varchar(20))")
	_, err = db.Exec("CREATE TABLE  IF NOT EXISTS presets (ID SERIAL PRIMARY KEY,OwnerID int, MemberID int, Permissions int)")
	_, err = db.Exec("CREATE TABLE  IF NOT EXISTS metanote (NoteID SERIAL PRIMARY KEY, MemberID int, Permissions int)")
	_, err = db.Exec("CREATE TABLE  IF NOT EXISTS note (ID SERIAL PRIMARY KEY, Note varchar(2550))")
	//_, err = db.Exec("INSERT INTO members (Password) VALUES ('password'),('bird'),('cat'),('dog'),('tree')")

	if err != nil {
		panic(err)
	}

	tpl = template.Must(template.ParseGlob("templates/*"))
}

func main() {
	http.HandleFunc("/", index)
	http.HandleFunc("/about", about)
	http.HandleFunc("/members", members)
	http.HandleFunc("/members/new", membersCreateForm)
	http.HandleFunc("/members/new/process", membersCreateProcess)
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

func members(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT * FROM members")

	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}
	defer rows.Close()

	mbrs := make([]Members, 0)
	for rows.Next() {
		mbr := Members{}
		err := rows.Scan(&mbr.ID, &mbr.Password)

		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			return
		}
		mbrs = append(mbrs, mbr)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	tpl.ExecuteTemplate(w, "members.gohtml", mbrs)
}

func membersCreateForm(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "create.gohtml", nil)
}

func membersCreateProcess(w http.ResponseWriter, r *http.Request) {
	mbr := Members{}
	mbr.Password = r.FormValue("password")

	_, err := db.Exec("INSERT INTO members (Password) VALUES ($1)", mbr.Password)
	if err != nil {
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}
	tpl.ExecuteTemplate(w, "create.gohtml", mbr)
}
