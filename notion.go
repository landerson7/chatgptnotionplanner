package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type RichText struct {
	Type string `json:"type"`
	Text struct {
		Content string `json:"content"`
	} `json:"text"`
}

type Paragraph struct {
	RichText []RichText `json:"rich_text"`
}

type ToDo struct {
	RichText []RichText `json:"rich_text"` // Add rich_text field here
	Checked  bool       `json:"checked"`
}

type Block struct {
	Object    string     `json:"object"`
	Type      string     `json:"type"`
	Paragraph *Paragraph `json:"paragraph,omitempty"`
	ToDo      *ToDo      `json:"to_do,omitempty"`
}

type Parent struct {
	PageID string `json:"page_id"`
}

type NotionRequest struct {
	Parent     Parent `json:"parent"`
	Properties struct {
		Title struct {
			Title []RichText `json:"title"`
		} `json:"title"`
	} `json:"properties"`
	Children []Block `json:"children"`
}

func SendToNotion(course string, to_do []string) {
	notionApiKey := GetEnvVar("NOTION_API")
	url := "https://api.notion.com/v1/pages"

	// Define the JSON structure using structs
	notionRequest := NotionRequest{
		Parent: Parent{
			PageID: "713ae619-b5cd-482f-a0c6-27b2fa1bf1dc",
		},
		Properties: struct {
			Title struct {
				Title []RichText `json:"title"`
			} `json:"title"`
		}{
			Title: struct {
				Title []RichText `json:"title"`
			}{
				Title: []RichText{
					{
						Type: "text",
						Text: struct {
							Content string `json:"content"`
						}{
							Content: course + " Assignments",
						},
					},
				},
			},
		},
		Children: []Block{
			{
				Object: "block",
				Type:   "paragraph",
				Paragraph: &Paragraph{
					RichText: []RichText{
						{
							Type: "text",
							Text: struct {
								Content string `json:"content"`
							}{
								Content: "Geology course to-dos retrieved from Webcourses",
							},
						},
					},
				},
			},
		},
	}

	// Add each to-do item as a new Block in the Children array
	for _, item := range to_do {
		toDoBlock := Block{
			Object: "block",
			Type:   "to_do",
			ToDo: &ToDo{
				RichText: []RichText{
					{
						Type: "text",
						Text: struct {
							Content string `json:"content"`
						}{
							Content: item,
						},
					},
				},
				Checked: false,
			},
		}
		notionRequest.Children = append(notionRequest.Children, toDoBlock)
	}

	sendData, err := json.Marshal(notionRequest)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(sendData))
	if err != nil {
		fmt.Println("error creating request:", err)
		return
	}

	req.Header.Add("Authorization", "Bearer "+notionApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	fmt.Println("Raw response body:", string(body))
}
