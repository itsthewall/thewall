package main

import (
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

var (
	addr  = flag.String("addr", "localhost:8080", "address to host the server on")
	dbURI = flag.String("db", "user=postgres password=password dbname=wall", "uri to access postgres database")
)

type User struct {
	ID    int64
	Name  string
	Email string
}

type Block struct {
	ID    int64
	Title string
}

type Post struct {
	ID int64

	BlockID int64
	UserID  int64

	Title string
	Body  string
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("templates/index.html", "templates/_layout.html"))

	type Data struct {
		Msg   string
		Posts []Post
	}

	data := Data{
		Posts: []Post{
			{Title: "hello world", Body: "goodbye!"},
			{Title: "hello world", Body: "goodbye!"},
			{Title: "hello world", Body: "goodbye!"},
		},
	}

	err := t.ExecuteTemplate(w, "_layout", data)
	if err != nil {
		fmt.Fprintln(w)
	}
}

func main() {
	flag.Parse()

	conn, err := sql.Open("postgres", *dbURI)
	if err != nil {
		log.Fatal(err)
	}

	err = conn.Ping()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/mail", handleMail)

	server := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
		Addr:         *addr,
	}

	log.Println("Starting server on", server.Addr)
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
