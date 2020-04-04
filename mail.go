package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	parsemail "github.com/DusanKasan/parsemail"
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
	defer w.WriteHeader(http.StatusOK)

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

	for _, embedded := range email.EmbeddedFiles {
		fmt.Println(embedded)
	}
	for _, attached := range email.Attachments {
		fmt.Println(attached)
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
		return nil, err
	}
	mr := multipart.NewReader(r.Body, params["boundary"])
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			return nil, errors.New("No email field found in body.")
		}
		if err != nil {
			return nil, err
		}
		if p.FormName() == "email" {
			return p, nil
		}
	}
}


// Outputs raw input to emails directory.
func saveAndReplaceReader(r io.Reader) (io.Reader, error) {
	file, err := os.Create(fmt.Sprintf("emails/%v", time.Now().Format(time.Stamp)))
	defer file.Close()
	if err != nil {
		fmt.Println(err)
		return r, nil
	}

	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	r = bytes.NewReader(raw)
	_, err = file.Write(raw)
	if err != nil {
		fmt.Println(err)
		return r, nil
	}
	return r, nil
}
