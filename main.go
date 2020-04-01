package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

var (
	addr = flag.String("addr", "localhost:8080", "address to host the server on")
)

func handleHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hello, world!")
}

func main() {
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHome)

	server := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
		Addr:         *addr,
	}

	log.Println("Starting server on", server.Addr)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
