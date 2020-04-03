package main

import (
	"database/sql"
	"errors"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"

	parsemail "github.com/DusanKasan/Parsemail"
)

func handleMail(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		io.WriteString(w, "Must use POST.")
		return
	}
	//Always respond with a 200 status code to prevent send grid from
	//resending emails.
	defer io.WriteString(w, "Received.")

	//TODO(obi) Verify that the request is coming from send grid.

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

	const userQuery string = "SELECT id, name, email FROM users WHERE email = $1;"
	user := User{}

	row := conn.QueryRow(userQuery, email.From[0].Address)
	if err := row.Scan(&user.ID, &user.Name, &user.Email); err == sql.ErrNoRows {
		log.Println("Email from unknown user: ", email.From[0].String())
		return
	} else if err != nil {
		log.Println("Database error: ", err)
		return
	}

	const currentBlock string = `SELECT id FROM blocks ORDER BY id DESC 
		FETCH FIRST ROW ONLY;`
	row = conn.QueryRow(currentBlock)
	var blockID int64
	if err := row.Scan(&blockID); err != nil {
		log.Println("Database error: ", err)
		return
	}

	post := Post{
		Title:   email.Subject,
		Body:    email.TextBody,
		UserID:  user.ID,
		BlockID: blockID,
	}

	const insertPost string = `INSERT INTO posts (block_id, user_id, title, body)
		VALUES ($1, $2, $3, $4);`
	_, err = conn.Exec(insertPost, post.BlockID, post.UserID, post.Title, post.Body)
	if err != nil {
		log.Println("Database insert error: ", err)
		return
	}

	log.Printf("Added a post to block %v by %v", post.BlockID, user.Name)
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
