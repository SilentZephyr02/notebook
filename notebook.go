package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

type Members struct {
	ID       int
	Username string
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

	_, err = db.Exec("DROP TABLE IF EXISTS members")
	_, err = db.Exec("DROP TABLE IF EXISTS presets")
	_, err = db.Exec("DROP TABLE IF EXISTS metanote")
	_, err = db.Exec("DROP TABLE IF EXISTS note")

	_, err = db.Exec("CREATE TABLE  IF NOT EXISTS members (ID SERIAL PRIMARY KEY,Username varchar(20) NOT NULL, Password varchar(20) NOT NULL, CONSTRAINT UC_Member UNIQUE (Username))")
	_, err = db.Exec("CREATE TABLE  IF NOT EXISTS presets (ID SERIAL PRIMARY KEY,OwnerID int, MemberID int, Permissions int)")
	_, err = db.Exec("CREATE TABLE  IF NOT EXISTS metanote (NoteID SERIAL PRIMARY KEY, MemberID int, Permissions int)")
	_, err = db.Exec("CREATE TABLE  IF NOT EXISTS note (ID SERIAL PRIMARY KEY, Note varchar(2550))")
	//_, err = db.Exec("INSERT INTO members (Username,Password) VALUES ('admin','password'),('John','bird'),('Cam','cat'),('Scott','dog'),('Leaf','tree')")
	//The above line allows us to generate a database with filled in fields for testing

	if err != nil {
		panic(err)
	}

	tpl = template.Must(template.ParseGlob("templates/*"))
}

func main() {
	http.HandleFunc("/", loginCreateForm)
	http.HandleFunc("/login", loginProcess)
	http.HandleFunc("/members", listAllMembers)
	http.HandleFunc("/members/new", membersCreateForm)
	http.HandleFunc("/members/new/process", membersCreateProcess)
	http.ListenAndServe(":8080", nil)
}

func loginCreateForm(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "login.gohtml", nil)
}

func loginProcess(w http.ResponseWriter, r *http.Request) {
	Username := r.FormValue("username")
	Password := r.FormValue("password")

	mbr := Members{}

	row := db.QueryRow("SELECT * FROM members WHERE USERNAME =$1 AND PASSWORD =$2", Username, Password)
	err := row.Scan(&mbr.ID, &mbr.Username, &mbr.Password)

	switch {
	case err == sql.ErrNoRows:
		//no row
		fmt.Println("No user found")
		tpl.ExecuteTemplate(w, "login.gohtml", "No User Found")
	case err != nil:
		//else error
		http.Error(w, http.StatusText(500), 500)
		return

	default:
		//1 row
		expiration := time.Now().Add(365 * 24 * time.Hour)
		cookieUsername := http.Cookie{Name: "username", Value: mbr.Username, Expires: expiration}
		cookieID := http.Cookie{Name: "ID", Value: strconv.Itoa(mbr.ID), Expires: expiration}
		http.SetCookie(w, &cookieUsername)
		http.SetCookie(w, &cookieID)

		tpl.ExecuteTemplate(w, "index.gohtml", mbr.Username)
	}
}

func listAllMembers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT * FROM members")

	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}
	defer rows.Close()

	mbrs := make([]Members, 0)
	for rows.Next() {
		mbr := Members{}
		err := rows.Scan(&mbr.ID, &mbr.Username, &mbr.Password)

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
	mbr.Username = r.FormValue("username")
	mbr.Password = r.FormValue("password")

	if checkIfMemberExists(mbr.Username) {
		fmt.Println("user exists")
		tpl.ExecuteTemplate(w, "create.gohtml", "exists")
		return
	}
	if mbr.Username == "" || mbr.Password == "" {
		fmt.Println("Field is nil")
		tpl.ExecuteTemplate(w, "create.gohtml", nil)
		return
	}

	_, err := db.Exec("INSERT INTO members (Username,Password) VALUES ($1,$2)", mbr.Username, mbr.Password)
	if err != nil {
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}
	tpl.ExecuteTemplate(w, "create.gohtml", mbr)
}

func checkIfMemberExists(name string) bool {
	rows, err := db.Query("SELECT COUNT(Username) FROM members WHERE Username=$1", name)
	rows.Next()
	var count int
	err = rows.Scan(&count)
	if err != nil {
		panic(err)
	}
	if count > 0 {
		return true
	}
	return false

}
