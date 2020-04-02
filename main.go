package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

var (
	addr  = flag.String("addr", "localhost:8080", "address to host the server on")
	dbURI = flag.String("db", "", "uri to access postgres database")
)

func handleHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hello, world!")
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
