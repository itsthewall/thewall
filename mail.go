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
	"regexp"
	"strings"
	"time"

	parsemail "github.com/DusanKasan/parsemail"
	markdown "github.com/gomarkdown/markdown"
)

const ImagesLocation string = "images/"

func handleMail(w http.ResponseWriter, r *http.Request) {
	log.Println("Mail request: ", r.Method, r.RemoteAddr)
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

	log.Println(email.TextBody)

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

	for _, attached := range email.Attachments {
		fmt.Println(attached)
	}

	const lastBlock string = `SELECT id, created_at FROM blocks ORDER BY created_at DESC FETCH FIRST ROW ONLY;`

	row = conn.QueryRow(lastBlock)
	var blockID int64
	var blockCreatedAt time.Time
	if err := row.Scan(&blockID, &blockCreatedAt); err != nil {
		log.Println("Database error: ", err)
		return
	}

	if blockCreatedAt.Before(time.Now().Add(-schedule.Frequency)) {
		delta := time.Now().Sub(blockCreatedAt)
		count := delta / schedule.Frequency
		const newBlock = `INSERT INTO blocks (title, created_at) VALUES ($1, $2) RETURNING id`
		idRow := conn.QueryRow(newBlock, "TODO(obi)", blockCreatedAt.Add(schedule.Frequency*count))

		err = idRow.Scan(&blockID)
		if err != nil {
			log.Println("Database error: ", err)
			return
		}
	}

	body := email.TextBody

	newlineKiller := strings.NewReplacer(
		"\r", "\n",
	)

	body = newlineKiller.Replace(body)

	// Convert markdown to HTML
	html := string(markdown.ToHTML([]byte(body), nil, nil))

	replacer, err := saveEmbedded(&email.EmbeddedFiles)
	if err != nil {
		log.Println(err)
	}

	html = replacer.Replace(html)

	re := regexp.MustCompile(`#(\d+)`)
	html = re.ReplaceAllString(html, `<a href="/post?id=$1">#$1</a>`)

	post := Post{
		Title:   email.Subject,
		Body:    html,
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

func saveEmbedded(files *[]parsemail.EmbeddedFile) (*strings.Replacer, error) {
	var replacementList []string
	for _, embedded := range *files {
		mediaType, params, err := mime.ParseMediaType(embedded.ContentType)

		name := params["name"]

		fileName := fmt.Sprint(ImagesLocation, embedded.CID, '-', name)

		img, err := ioutil.ReadAll(embedded.Data)
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(fileName, img, os.FileMode(0400))
		if err != nil {
			return nil, err
		}

		const textTag, htmlTag string = "[%v: %v]", "<%v src=\"/%v\">"

		switch strings.Split(mediaType, "/")[0] {
		case "image":
			replacementList = append(replacementList,
				fmt.Sprintf(textTag, "image", name),
				fmt.Sprintf(htmlTag, "img", fileName))
		}
	}
	return strings.NewReplacer(replacementList...), nil
}

// Outputs raw input to emails directory.
// Used to debug parser/save test emails for later.
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
