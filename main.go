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

var (
	conn *sql.DB
)

type User struct {
	ID    int64
	Name  string
	Email string
}

type Block struct {
	ID    int64
	Title string
	Posts []Post
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
		Msg    string
		Blocks []Block
	}

	const blocksQuery = `SELECT id, title FROM blocks ORDER BY id DESC;`

	blocks, err := conn.Query(blocksQuery)
	if err != nil {
		// TODO(harrison): don't send DB errors dummy to the user dummy!
		fmt.Fprintln(w, err)

		return
	}

	data := Data{}

	for blocks.Next() {
		b := Block{}
		if err := blocks.Scan(&b.ID, &b.Title); err != nil {
			// TODO(harrison): don't send DB errors to users!
			fmt.Fprintln(w, err)

			return
		}

		const postQuery = `SELECT id, block_id, user_id, title, body FROM posts WHERE block_id = $1`
		posts, err := conn.Query(postQuery, b.ID)

		for posts.Next() {
			p := Post{}

			if err = posts.Scan(&p.ID, &p.BlockID, &p.UserID, &p.Title, &p.Body); err != nil {
				// TODO(harrison): BAD.
				fmt.Fprintln(w, err)

				return
			}

			b.Posts = append(b.Posts, p)
		}

		data.Blocks = append(data.Blocks, b)
	}

	err = t.ExecuteTemplate(w, "_layout", data)
	if err != nil {
		fmt.Fprintln(w, err)
	}
}

func main() {
	var err error
	flag.Parse()

	conn, err = sql.Open("postgres", *dbURI)
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

	//Serve static files
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	mux.Handle("/static/", staticHandler)

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
