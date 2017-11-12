package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
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

type Permissions struct {
	Username   string
	Permission int
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
*/
type Note struct {
	ID   int
	Note string
}

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
	_, err = db.Exec("CREATE TABLE  IF NOT EXISTS metanote (NoteID int, MemberID int, Permissions int, PRIMARY KEY(NoteID, MemberID))")
	_, err = db.Exec("CREATE TABLE  IF NOT EXISTS note (ID SERIAL PRIMARY KEY, Note varchar(2550))")
	_, err = db.Exec("INSERT INTO members (Username,Password) VALUES ('admin','password'),('John','bird'),('Cam','cat'),('Scott','dog'),('Leaf','tree')")
	//The above line allows us to generate a database with filled in fields for testing

	if err != nil {
		panic(err)
	}

	tpl = template.Must(template.ParseGlob("templates/*"))
}

func main() {
	http.HandleFunc("/", loginCreateForm)
	http.HandleFunc("/login", loginProcess)
	http.HandleFunc("/logout", logoutProcess)

	http.HandleFunc("/members", listAllMembers)
	http.HandleFunc("/members/new", membersCreateForm)
	http.HandleFunc("/members/new/process", membersCreateProcess)
	http.HandleFunc("/members/update", membersUpdateForm)
	http.HandleFunc("/members/update/process", membersUpdateProcess)
	http.HandleFunc("/members/delete", membersDeleteProcess)

	http.HandleFunc("/note/list", listOwnersNotes)
	http.HandleFunc("/note/new", noteCreateForm)
	http.HandleFunc("/note/new/process", noteCreateProcess)
	http.HandleFunc("/note/update", noteUpdateForm)
	http.HandleFunc("/note/update/process", noteUpdateProcess)
	http.HandleFunc("/note/delete", noteDeleteForm)
	http.HandleFunc("/note/permissions", notePermissionsForm)
	http.ListenAndServe(":8080", nil)
}

func notePermissionsForm(w http.ResponseWriter, r *http.Request) {
	//inner join to get all users with permissions to this note,
	// list and allow edit of permissions & addition of read access
	noteID := r.FormValue("id")
	if noteID == "" {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}
	rows, err := db.Query("SELECT members.username, metanote.permissions FROM members INNER JOIN metanote ON members.ID = metanote.memberid INNER JOIN note ON metanote.noteid = note.id WHERE metanote.noteid = $1", noteID)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}
	users := make([]Permissions, 0)
	for rows.Next() {
		user := Permissions{}
		rowErr := rows.Scan(&user.Username, &user.Permission)

		if rowErr != nil {
			http.Error(w, http.StatusText(500), 500)
			return
		}
		users = append(users, user)
	}
	tpl.ExecuteTemplate(w, "permissions.gohtml", users)

}

func noteCreateForm(w http.ResponseWriter, r *http.Request) {
	if loggedInCheck(r) {
		tpl.ExecuteTemplate(w, "noteCreateForm.gohtml", nil)
	} else {
		tpl.ExecuteTemplate(w, "login.gohtml", "You must be logged in to continue")
	}
}

func loggedInCheck(r *http.Request) bool {
	_, err := r.Cookie("ID")
	if err == http.ErrNoCookie {
		return false
	} else if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func getCurrentID(r *http.Request) int {
	cookieID, _ := r.Cookie("ID")
	ID, _ := strconv.Atoi(cookieID.Value)
	return ID
}

func getCurrentUsername(r *http.Request) string {
	cookieUser, _ := r.Cookie("Username")
	Username := cookieUser.Value
	return Username
}

func noteCreateProcess(w http.ResponseWriter, r *http.Request) {
	if loggedInCheck(r) {
		_, err := db.Exec("INSERT INTO note (Note) VALUES ($1)", r.FormValue("message"))
		if err != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}
		cookie, _ := r.Cookie("ID")
		s := cookie.Value
		memberID, _ := strconv.Atoi(s)
		per := 111
		noteID := 0
		result := db.QueryRow("SELECT ID FROM note ORDER BY ID DESC LIMIT 1")
		newErr := result.Scan(&noteID)
		switch {
		case newErr == sql.ErrNoRows:
			//no row
			fmt.Println("No user found")
			tpl.ExecuteTemplate(w, "login.gohtml", "No User Found")
		case newErr != nil:
			//else error
			http.Error(w, http.StatusText(500), 500)
			return
		default:
			addMetaNote(noteID, memberID, per)
			listOwnersNotes(w, r)
		}
	} else {
		tpl.ExecuteTemplate(w, "login.gohtml", "You must be logged in to continue")
	}
}

func addMetaNote(noteID int, memberID int, per int) {
	_, err := db.Exec("INSERT INTO metanote (NoteID, memberID, permissions) VALUES ($1, $2, $3)", noteID, memberID, per)
	if err != nil {
		panic(err)
	}
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

func logoutProcess(w http.ResponseWriter, r *http.Request) {
	expiredCookie := &http.Cookie{Name: "username", MaxAge: -10, Expires: time.Now()}
	http.SetCookie(w, expiredCookie)
	expiredCookie = &http.Cookie{Name: "ID", MaxAge: -10, Expires: time.Now()}
	http.SetCookie(w, expiredCookie)

	loginCreateForm(w, r)
}

func listAllMembers(w http.ResponseWriter, r *http.Request) {
	if loggedInCheck(r) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
			return
		}

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
	} else {
		tpl.ExecuteTemplate(w, "login.gohtml", "You must be logged in to continue")
	}
}

func listOwnersNotes(w http.ResponseWriter, r *http.Request) {
	if loggedInCheck(r) {
		rows, err := db.Query("select id,note from note inner join metanote on note.id = noteid where memberid = $1", getCurrentID(r))

		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			return
		}
		defer rows.Close()

		notes := make([]Note, 0)
		for rows.Next() {
			note := Note{}
			err := rows.Scan(&note.ID, &note.Note)

			if err != nil {
				http.Error(w, http.StatusText(500), 500)
				return
			}
			notes = append(notes, note)
		}

		if err = rows.Err(); err != nil {
			http.Error(w, http.StatusText(500), 500)
			return
		}

		tpl.ExecuteTemplate(w, "notes.gohtml", notes)
	} else {
		tpl.ExecuteTemplate(w, "login.gohtml", "You must be logged in to continue")
	}
}

func membersCreateForm(w http.ResponseWriter, r *http.Request) {
	if loggedInCheck(r) {
		tpl.ExecuteTemplate(w, "create.gohtml", nil)
	} else {
		tpl.ExecuteTemplate(w, "login.gohtml", "You must be logged in to continue")
	}
}

func membersCreateProcess(w http.ResponseWriter, r *http.Request) {
	if loggedInCheck(r) {
		if r.Method != "POST" {
			http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
			return
		}

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
	} else {
		tpl.ExecuteTemplate(w, "login.gohtml", "You must be logged in to continue")
	}
}

func membersUpdateForm(w http.ResponseWriter, r *http.Request) {
	if loggedInCheck(r) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
			return
		}

		memberID := r.FormValue("id")
		if memberID == "" {
			http.Error(w, http.StatusText(400), http.StatusBadRequest)
			return
		}

		row := db.QueryRow("SELECT * FROM members WHERE id = $1", memberID)

		mbr := Members{}
		err := row.Scan(&mbr.ID, &mbr.Username, &mbr.Password)
		switch {
		case err == sql.ErrNoRows:
			http.NotFound(w, r)
			return
		case err != nil:
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}
		tpl.ExecuteTemplate(w, "update.gohtml", mbr)
	} else {
		tpl.ExecuteTemplate(w, "login.gohtml", "You must be logged in to continue")
	}
}

func noteUpdateForm(w http.ResponseWriter, r *http.Request) {
	if loggedInCheck(r) {
		/*
			if r.Method != "GET" {
				http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
				return
			}
		*/

		noteID := r.FormValue("id")
		if noteID == "" {
			http.Error(w, http.StatusText(400), http.StatusBadRequest)
			return
		}

		row := db.QueryRow("SELECT * FROM note WHERE id = $1", noteID)

		note := Note{}
		err := row.Scan(&note.ID, &note.Note)
		switch {
		case err == sql.ErrNoRows:
			http.NotFound(w, r)
			return
		case err != nil:
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}
		tpl.ExecuteTemplate(w, "noteUpdate.gohtml", note)
	} else {
		tpl.ExecuteTemplate(w, "login.gohtml", "You must be logged in to continue")
	}
}

func membersUpdateProcess(w http.ResponseWriter, r *http.Request) {
	if loggedInCheck(r) {
		if r.Method != "POST" {
			http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
			return
		}

		memberID := r.FormValue("id")
		if memberID == "" {
			http.Error(w, http.StatusText(400), http.StatusBadRequest)
			return
		}

		mbr := Members{}
		mbr.Username = r.FormValue("Username")
		mbr.Password = r.FormValue("Password")

		_, err := db.Exec("UPDATE members SET Username=$1,Password=$2 WHERE ID=$3", mbr.Username, mbr.Password, memberID)
		if err != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}

		listAllMembers(w, r)
	} else {
		tpl.ExecuteTemplate(w, "login.gohtml", "You must be logged in to continue")
	}
}

func noteUpdateProcess(w http.ResponseWriter, r *http.Request) {
	if loggedInCheck(r) {
		if r.Method != "POST" {
			http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
			return
		}

		noteID := r.FormValue("id")
		if noteID == "" {
			http.Error(w, http.StatusText(400), http.StatusBadRequest)
			return
		}

		note := Note{}
		note.Note = r.FormValue("message")

		_, err := db.Exec("UPDATE note SET Note=$1 WHERE ID=$2", note.Note, noteID)
		if err != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}

		listOwnersNotes(w, r)
	} else {
		tpl.ExecuteTemplate(w, "login.gohtml", "You must be logged in to continue")
	}
}

func membersDeleteProcess(w http.ResponseWriter, r *http.Request) {
	if loggedInCheck(r) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
			return
		}

		memberID := r.FormValue("id")
		if memberID == "" {
			http.Error(w, http.StatusText(400), http.StatusBadRequest)
			return
		}

		_, err := db.Exec("DELETE FROM members WHERE ID=$1;", memberID)
		if err != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}

		listAllMembers(w, r)
	} else {
		tpl.ExecuteTemplate(w, "login.gohtml", "You must be logged in to continue")
	}
}

func noteDeleteForm(w http.ResponseWriter, r *http.Request) {
	if loggedInCheck(r) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
			return
		}

		noteID := r.FormValue("id")
		if noteID == "" {
			http.Error(w, http.StatusText(400), http.StatusBadRequest)
			return
		}

		_, err := db.Exec("DELETE FROM note WHERE ID=$1;", noteID)
		if err != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}

		listOwnersNotes(w, r)
	} else {
		tpl.ExecuteTemplate(w, "login.gohtml", "You must be logged in to continue")
	}
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
