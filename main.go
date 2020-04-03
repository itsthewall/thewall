package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

const tokenSize = 64

var (
	addr     = flag.String("addr", "localhost:8080", "address to host the server on")
	dbURI    = flag.String("db", "", "uri to access postgres database")
	password = flag.String("password", "", "password for access to the server")
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

	Time time.Time
}

type Post struct {
	ID int64

	BlockID int64
	UserID  int64

	Title string
	Body  string

	Time time.Time
}

var funcMap template.FuncMap = template.FuncMap{
	"Format": time.Time.Format,
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("index").Funcs(funcMap).ParseFiles("templates/index.html", "templates/_layout.html"))

	type PostInfo struct {
		Post
		ByUser string
	}

	type BlockInfo struct {
		Block
		Posts []PostInfo
	}

	type Data struct {
		Blocks []BlockInfo
	}

	const blocksQuery = `SELECT id, title FROM blocks ORDER BY id DESC;`

	blocks, err := conn.Query(blocksQuery)
	if err != nil {
		// TODO(harrison): don't send DB errors to the user dummy!
		fmt.Fprintln(w, err)

		return
	}

	data := Data{}

	for blocks.Next() {
		bi := BlockInfo{}
		if err := blocks.Scan(&bi.ID, &bi.Title); err != nil {
			// TODO(harrison): don't send DB errors to users!
			fmt.Fprintln(w, err)

			return
		}

		const postQuery = `
SELECT
	posts.id, posts.block_id, posts.user_id, posts.title, posts.body, posts.created_at, users.name
FROM
	posts, users
WHERE
	posts.user_id = users.id AND block_id = $1
`
		posts, err := conn.Query(postQuery, bi.ID)
		if err != nil {
			// TODO(harrison): don't send DB errors to the user dummy!
			fmt.Fprintln(w, err)

			return
		}

		for posts.Next() {
			pi := PostInfo{}

			if err = posts.Scan(&pi.ID, &pi.BlockID, &pi.UserID, &pi.Title, &pi.Body, &pi.Time, &pi.ByUser); err != nil {
				// TODO(harrison): BAD.
				fmt.Fprintln(w, err)

				return
			}

			bi.Posts = append(bi.Posts, pi)
		}

		data.Blocks = append(data.Blocks, bi)
	}

	err = t.ExecuteTemplate(w, "_layout", data)
	if err != nil {
		fmt.Fprintln(w, err)
	}
}

func handlePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		type Data struct {
			DidError bool
		}

		t := template.Must(template.ParseFiles("templates/password.html", "templates/_layout.html"))

		err := t.ExecuteTemplate(w, "_layout", Data{DidError: r.URL.Query().Get("error") == "true"})
		if err != nil {
			fmt.Fprintln(w, err)
		}

		return
	}

	r.ParseForm()

	if r.FormValue("password") != *password {
		http.Redirect(w, r, "/password?error=true", http.StatusFound)

		return
	}

	blk := make([]byte, tokenSize)
	_, err := rand.Read(blk)
	if err != nil {
		// TODO(harrison): handle errors properly
		fmt.Fprintln(w, err)

		return
	}

	token := hex.EncodeToString(blk)

	query := `INSERT INTO tokens (token) VALUES ($1);`
	_, err = conn.Exec(query, token)
	if err != nil {
		// TODO(harrison): handle errors properly
		fmt.Fprintln(w, err)

		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "Auth",
		Value: token,
		// NOTE(harrison): Commented this out because we don't actually enforce it in the DB
		// Expires: time.Now().AddDate(0, 0, 7),
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func authenticateOr(f http.HandlerFunc, or string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, c := range r.Cookies() {
			if c.Name != "Auth" {
				continue
			}

			query := `SELECT id FROM tokens WHERE token = $1`
			row := conn.QueryRow(query, c.Value)

			var id int64
			err := row.Scan(&id)

			if err == sql.ErrNoRows {
				fmt.Fprintln(w, "auth error. sorry!")

				break
			} else if err != nil {
				// TODO(harrison): handle actual error
				fmt.Fprintln(w, err)

				return
			}

			f(w, r)

			return
		}

		http.Redirect(w, r, or, http.StatusFound)
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

	if err := migrate(conn); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	// User routes
	mux.HandleFunc("/", authenticateOr(handleHome, "/password"))
	mux.HandleFunc("/password", handlePassword)

	// API routes
	mux.HandleFunc("/mail", handleMail)

	// Serve static files
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
