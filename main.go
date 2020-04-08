package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"flag"
	"html/template"
	"log"
	"net/http"
	"strconv"
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

type PostInfo struct {
	Post
	HTMLBody template.HTML
	ByUser   string
}

type BlockInfo struct {
	Block
	Posts []PostInfo
}

func handleHome(w http.ResponseWriter, r *http.Request) *Error {
	t := template.Must(template.New("index").Funcs(funcMap).ParseFiles("templates/index.html", "templates/_layout.html", "templates/_post.html"))

	type Data struct {
		Blocks []BlockInfo
	}

	const blocksQuery = `SELECT id, title FROM blocks ORDER BY id DESC;`

	blocks, err := conn.Query(blocksQuery)
	if err != nil {
		return ErrorForDatabase(err)
	}

	data := Data{}

	for blocks.Next() {
		bi := BlockInfo{}
		if err := blocks.Scan(&bi.ID, &bi.Title); err != nil {
			return ErrorForDatabase(err)
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
			return ErrorForDatabase(err)
		}

		for posts.Next() {
			pi := PostInfo{}

			if err = posts.Scan(&pi.ID, &pi.BlockID, &pi.UserID, &pi.Title, &pi.HTMLBody, &pi.Time, &pi.ByUser); err != nil {
				return ErrorForDatabase(err)
			}

			bi.Posts = append(bi.Posts, pi)
		}

		data.Blocks = append(data.Blocks, bi)
	}

	err = t.ExecuteTemplate(w, "_layout", data)
	if err != nil {
		return &Error{
			Err:     err,
			Message: "template rendering error",
			Code:    http.StatusInternalServerError,
		}
	}

	return nil
}

func handlePost(w http.ResponseWriter, r *http.Request) *Error {
	r.ParseForm()

	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		return &Error{
			Err:     err,
			Message: "not a valid id",
			Code:    http.StatusInternalServerError,
		}
	}

	const postQuery = `
SELECT
	posts.id, posts.block_id, posts.user_id, posts.title, posts.body, posts.created_at, users.name
FROM
	posts, users
WHERE
	posts.user_id = users.id AND posts.id = $1
`
	row := conn.QueryRow(postQuery, id)
	if err != nil {
		return ErrorForDatabase(err)
	}

	pi := PostInfo{}

	if err = row.Scan(&pi.ID, &pi.BlockID, &pi.UserID, &pi.Title, &pi.HTMLBody, &pi.Time, &pi.ByUser); err != nil {
		return ErrorForDatabase(err)
	}

	type Data struct {
		Post PostInfo
	}

	t := template.Must(template.New("post").Funcs(funcMap).ParseFiles("templates/post.html", "templates/_layout.html", "templates/_post.html"))
	err = t.ExecuteTemplate(w, "_layout", Data{Post: pi})
	if err != nil {
		return &Error{
			Err:     err,
			Message: "template rendering error",
			Code:    http.StatusInternalServerError,
		}
	}

	return nil
}

func handlePassword(w http.ResponseWriter, r *http.Request) *Error {
	if r.Method == "GET" {
		type Data struct {
			DidError bool
		}

		t := template.Must(template.ParseFiles("templates/password.html", "templates/_layout.html"))

		err := t.ExecuteTemplate(w, "_layout", Data{DidError: r.URL.Query().Get("error") == "true"})
		if err != nil {
			return &Error{
				Err:     err,
				Message: "template rendering error",
				Code:    http.StatusInternalServerError,
			}
		}

		return nil
	}

	r.ParseForm()

	if r.FormValue("password") != *password {
		http.Redirect(w, r, "/password?error=true", http.StatusFound)

		return nil
	}

	blk := make([]byte, tokenSize)
	_, err := rand.Read(blk)
	if err != nil {
		return &Error{
			Err:     err,
			Message: "random generation error",
			Code:    http.StatusInternalServerError,
		}
	}

	token := hex.EncodeToString(blk)

	query := `INSERT INTO tokens (token) VALUES ($1);`
	_, err = conn.Exec(query, token)
	if err != nil {
		return ErrorForDatabase(err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "Auth",
		Value: token,
		// NOTE(harrison): Commented this out because we don't actually enforce it in the DB
		// Expires: time.Now().AddDate(0, 0, 7),
	})

	http.Redirect(w, r, "/", http.StatusFound)

	return nil
}

func authenticateOr(f ErrorHandler, or string) ErrorHandler {
	return func(w http.ResponseWriter, r *http.Request) *Error {
		for _, c := range r.Cookies() {
			if c.Name != "Auth" {
				continue
			}

			query := `SELECT id FROM tokens WHERE token = $1`
			row := conn.QueryRow(query, c.Value)

			var id int64
			err := row.Scan(&id)

			if err == sql.ErrNoRows {
				// TODO(harrison): this should probably just redirect...
				return &Error{
					Err:     err,
					Message: "auth token doesn't exist. log in again.",
					Code:    http.StatusInternalServerError,
				}

				break
			} else if err != nil {
				return ErrorForDatabase(err)
			}

			f(w, r)

			return nil
		}

		http.Redirect(w, r, or, http.StatusFound)

		return nil
	}
}

type Error struct {
	Err     error
	Message string
	Code    int
}

type ErrorHandler func(w http.ResponseWriter, r *http.Request) *Error

func (eh ErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e := eh(w, r)

	if e == nil {
		return
	}

	log.Println("HTTP error:", e.Err)

	http.Error(w, e.Message, e.Code)
}

func ErrorForDatabase(err error) *Error {
	return &Error{
		Err:     err,
		Message: "database error",
		Code:    http.StatusInternalServerError,
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
	mux.Handle("/", ErrorHandler(authenticateOr(handleHome, "/password")))
	mux.Handle("/post", ErrorHandler(authenticateOr(handlePost, "/password")))
	mux.Handle("/password", ErrorHandler(handlePassword))

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
