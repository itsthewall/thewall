package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sendgrid/sendgrid-go"
)

type SendGridEmail struct {
	Personalizations []Personalizations `json:"personalizations"`
	TemplateID       string             `json:"template_id"`
	From             ToField            `json:"from"`
	ASM              ASM                `json:"asm"`
}

type Personalizations struct {
	To                  []ToField           `json:"to"`
	DynamicTemplateData DynamicTemplateData `json:"dynamic_template_data"`
}

type ToField struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type DynamicTemplateData struct {
	BlockInfo
	DateStamp string
	Subject   int64 `json:"subject"`
}

type ASM struct {
	GroupID int `json:"group_id"`
}

func sendDailyUpdate(block BlockInfo, to []string) error {

	tos := make([]ToField, len(to))
	for i, add := range to {
		tos[i] = ToField{Email: add}
	}

	email := SendGridEmail{
		From: ToField{
			Email: "digest@wall.obanana.rocks",
			Name:  "The Wall Digest",
		},
		Personalizations: []Personalizations{{
			To: tos,
			DynamicTemplateData: DynamicTemplateData{
				block,
				block.Time.Local().Format("Mon Jan 2 2006"),
				block.ID,
			},
		}},
		TemplateID: os.Getenv("SENDGRID_TEMPLATE_ID"),
		ASM: ASM{
			GroupID: 14556,
		},
	}

	json, err := json.MarshalIndent(email, "", "  ")
	if err != nil {
		return err
	}

	request := sendgrid.GetRequest(os.Getenv("SENDGRID_API_KEY"), "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	request.Body = json

	fmt.Println(string(json))

	response, err := sendgrid.MakeRequest(request)

	if err != nil {
		return err
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}

	return nil
}
