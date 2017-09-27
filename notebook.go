package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

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

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("path", r.URL.Path)
	fmt.Println(r.URL.Query())

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	myItems := []string{"item1", "item2", "item3"}
	a, _ := json.Marshal(myItems)

	w.Write(a)
	return
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
