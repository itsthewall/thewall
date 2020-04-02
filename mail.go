package main

import (
	"io"
	"net/http"
	"os"
)

func handleMail(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		io.WriteString(w, "Must use POST.")
		return
	}

	io.Copy(os.Stdout, r.Body)

	w.WriteHeader(200)
	io.WriteString(w, "OK")
}
