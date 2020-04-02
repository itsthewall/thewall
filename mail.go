package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/mail"

	parsemail "github.com/DusanKasan/Parsemail"
)

type Post struct { //Simple representation of emails for now.
	Title string
	Body  string
	Email *mail.Address
}

func handleMail(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		io.WriteString(w, "Must use POST.")
		return
	}
	//Always respond with a 200 status code to prevent send grid from
	//resending emails.
	defer w.WriteHeader(200)
	defer io.WriteString(w, "Received.")

	//TODO Verify that the request is coming from send grid.

	rawEmail, err := getRawEmail(r)
	if err != nil {
		log.Println(err)
		return
	}
	email, err := parsemail.Parse(rawEmail)
	if err != nil {
		log.Printf("Error parsing email: %v", err)
		return
	}

	if len(email.From) == 0 {
		return
	}

	post := Post{
		Title: email.Subject,
		Body:  email.TextBody,
		Email: email.From[0],
	}

	fmt.Printf("%v: %v\n%v", post.Title, post.Email.String(), post.Body)
}

func getRawEmail(r *http.Request) (io.Reader, error) {
	//Parse Grid gives us a multipart encoded POST request. Which we need to extract
	//the raw email from.
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		log.Println(err)
	}
	mr := multipart.NewReader(r.Body, params["boundary"])
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			return nil, errors.New("No email field found in body.")
		}
		if err != nil {
			log.Fatal(err)
		}
		if p.FormName() == "email" {
			return p, nil
		}
	}
}
