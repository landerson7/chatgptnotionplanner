package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type RichText struct {
	Type string `json:"type"`
	Text struct {
		Content string `json:"content"`
	} `json:"text"`
	Annotations struct {
		Bold bool `json:"bold"`
	} `json:"annotations"`
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

func SendAllAssignmentsToNotion() {
	var courses = map[string]int{
		"Geology":  1464091,
		"Cinema":   1463455,
		"CompComm": 1464602,
		"E1Lab":    1465496,
		"E1Lec":    1465493,
		"OS":       1464092,
	}

	//canvasApiKey := GetEnvVar("CANVAS_API")
	var assignments []assignment_due
	var discussions []discussion_due
	for courseName, course := range courses {

		assignments = GetAllAssignmentsByCourse(course)
		discussions = GetDiscussionPostByCourse(course)

		todos := []string{}
		dt := time.Now()

		if len(assignments) != 0 {
			for _, discussion := range assignments {
				// Parse the Due_At string into a time.Time object
				dueAtTime, err := time.Parse(time.RFC3339, discussion.Due_At)
				if err != nil {
					fmt.Println("Error parsing time:", err)
					continue
				}

				// Format the Due_At time
				todo := formatTime(discussion.Due_At)
				// Check if the current time is not before the due date
				if dt.Before(dueAtTime) { // && !discussion.Locked_For_User {
					fmt.Print("executed\n")
					todo = "Assignment: " + discussion.Name + " Due at: " + todo
					todos = append(todos, todo)
				}
			}
		}

		if len(discussions) != 0 {
			for _, discussion := range discussions {
				// Parse the Assignment's Due_At string into a time.Time object
				dueAtTime, err := time.Parse(time.RFC3339, discussion.Assignment.Due_At)
				if err != nil {
					fmt.Println("Error parsing time:", err)
					continue
				}

				// Format the Due_At time
				todo := formatTime(discussion.Assignment.Due_At)
				// Check if the current time is not before the due date
				if dt.Before(dueAtTime) { // && !discussion.Assignment.Locked_For_User {
					fmt.Print("executed\n")
					todo = "Assignment: " + discussion.Title + " Due at: " + todo
					todos = append(todos, todo)
				}

			}
		}

		SendToNotion(courseName+" Assignments as of "+FormatDate(dt), todos)
	}
	//updateToDoList("cdf832e3-454f-47cf-ab04-d2d63d4a6e00", todos)
}

/*
func SendAllAssignmentsToOneNotionPage() {
	var courses = map[string]int{
		"Geology":  1461901,
		"Cinema":   1463455,
		"CompComm": 1464602,
		"E1Lab":    1465496,
		"E1Lec":    1465493,
		"OS":       1464092,
	}

	notionApiKey := GetEnvVar("NOTION_API")
	url := "https://api.notion.com/v1/pages"

	// Initialize the Notion request with the parent and properties
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
							Content: FormatDate(time.Now()) + " Assignments and Discussions Due Within a Month",
						},
					},
				},
			},
		},
		Children: []Block{},
	}

	now := time.Now()
	oneMonthLater := now.AddDate(0, 1, 0)

	for courseName, courseID := range courses {
		assignments := GetAllAssignmentsByCourse(courseID)
		discussions := GetDiscussionPostByCourse(courseID)
		if courseName == "Geology" {
			assignments = GetAllAssignmentsByModule()
		}
		// Add a paragraph block for each course
		notionRequest.Children = append(notionRequest.Children, Block{
			Object: "block",
			Type:   "paragraph",
			Paragraph: &Paragraph{
				RichText: []RichText{
					{
						Type: "text",
						Text: struct {
							Content string `json:"content"`
						}{
							Content: courseName + " Assignments and Discussions",
						},
						Annotations: struct {
							Bold bool `json:"bold"`
						}{
							Bold: true,
						},
					},
				},
			},
		})

		todos := []string{}

		if len(assignments) != 0 {
			for _, assignment := range assignments {
				dueAtTime, err := time.Parse(time.RFC3339, assignment.Due_At)
				if err != nil {
					fmt.Println("Error parsing time:", err)
					continue
				}
				if now.Before(dueAtTime) && dueAtTime.Before(oneMonthLater) {
					todo := "Assignment: " + assignment.Name + " Due at: " + formatTime(assignment.Due_At)
					todos = append(todos, todo)
				}
			}
		}

		if len(discussions) != 0 {
			for _, discussion := range discussions {
				dueAtTime, err := time.Parse(time.RFC3339, discussion.Assignment.Due_At)
				fmt.Println(courseName + discussion.Title + discussion.Assignment.Due_At)
				if err != nil {
					fmt.Println("Error parsing time:", err)
					continue
				}
				if now.Before(dueAtTime) && dueAtTime.Before(oneMonthLater) {
					todo := "Discussion: " + discussion.Title + " Due at: " + formatTime(discussion.Assignment.Due_At)
					fmt.Println("discussion: " + todo)
					todos = append(todos, todo)
				}
			}
		}

		// Add each to-do item as a new Block in the Children array
		for _, item := range todos {
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
}*/

type Course struct {
	Name     string
	CourseID int
}

func SendAllAssignmentsToOneNotionPage() NotionRequest {
	courses := []Course{
		{"Geology", 1461901},
		{"Cinema", 1463455},
		{"CompComm", 1464602},
		{"E1Lab", 1465496},
		{"E1Lec", 1465493},
		{"OS", 1464092},
	}

	notionApiKey := GetEnvVar("NOTION_API")
	url := "https://api.notion.com/v1/pages"

	// Initialize the Notion request with the parent and properties
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
							Content: FormatDate(time.Now()) + " Assignments and Discussions Due Within a Month",
						},
					},
				},
			},
		},
		Children: []Block{},
	}

	now := time.Now()
	oneMonthLater := now.AddDate(0, 1, 0)

	for _, course := range courses {
		assignments := GetAllAssignmentsByCourse(course.CourseID)
		discussions := GetDiscussionPostByCourse(course.CourseID)
		if course.Name == "Geology" {
			assignments = GetAllAssignmentsByModule()
		}
		// Add a paragraph block for each course
		notionRequest.Children = append(notionRequest.Children, Block{
			Object: "block",
			Type:   "paragraph",
			Paragraph: &Paragraph{
				RichText: []RichText{
					{
						Type: "text",
						Text: struct {
							Content string `json:"content"`
						}{
							Content: course.Name + " Assignments and Discussions",
						},
						Annotations: struct {
							Bold bool `json:"bold"`
						}{
							Bold: true,
						},
					},
				},
			},
		})

		todos := []string{}

		if len(assignments) != 0 {
			for _, assignment := range assignments {
				dueAtTime, err := time.Parse(time.RFC3339, assignment.Due_At)
				if err != nil {
					fmt.Println("Error parsing time:", err)
					continue
				}
				if now.Before(dueAtTime) && dueAtTime.Before(oneMonthLater) {
					todo := "Assignment: " + assignment.Name + " Due at: " + formatTime(assignment.Due_At)
					todos = append(todos, todo)
				}
			}
		}

		if len(discussions) != 0 {
			for _, discussion := range discussions {
				dueAtTime, err := time.Parse(time.RFC3339, discussion.Assignment.Due_At)
				fmt.Println(course.Name + discussion.Title + discussion.Assignment.Due_At)
				if err != nil {
					fmt.Println("Error parsing time:", err)
					continue
				}
				if now.Before(dueAtTime) && dueAtTime.Before(oneMonthLater) {
					todo := "Discussion: " + discussion.Title + " Due at: " + formatTime(discussion.Assignment.Due_At)
					fmt.Println("discussion: " + todo)
					todos = append(todos, todo)
				}
			}
		}

		// Add each to-do item as a new Block in the Children array
		for _, item := range todos {
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
	}

	sendData, err := json.Marshal(notionRequest)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return notionRequest
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(sendData))
	if err != nil {
		fmt.Println("error creating request:", err)
		return notionRequest
	}

	req.Header.Add("Authorization", "Bearer "+notionApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return notionRequest
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return notionRequest
	}
	fmt.Println("Raw response body:", string(body))
	return notionRequest
}

func DeleteNotionPage(pageID string) {
	notionApiKey := GetEnvVar("NOTION_API")
	url := "https://api.notion.com/v1/pages/" + pageID

	// Create the JSON body to archive the page
	bodyData := map[string]bool{
		"archived": true,
	}
	sendData, err := json.Marshal(bodyData)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(sendData))
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

// Define the structure for the search request
type NotionSearchRequest struct {
	Query  string `json:"query"`
	Filter struct {
		Value    string `json:"value"`
		Property string `json:"property"`
	} `json:"filter"`
}

// Define the structure for the search response
type NotionSearchResponse struct {
	Results []struct {
		ID string `json:"id"`
	} `json:"results"`
}

func ArchivePageByName(pageName string) {
	notionApiKey := GetEnvVar("NOTION_API")
	searchURL := "https://api.notion.com/v1/search"

	// Create the search request
	searchRequest := NotionSearchRequest{
		Query: pageName,
	}
	searchRequest.Filter.Value = "page"
	searchRequest.Filter.Property = "object"

	// Marshal the search request to JSON
	searchData, err := json.Marshal(searchRequest)
	if err != nil {
		fmt.Println("Error marshaling search request:", err)
		return
	}

	// Send the search request
	req, err := http.NewRequest("POST", searchURL, bytes.NewBuffer(searchData))
	if err != nil {
		fmt.Println("Error creating search request:", err)
		return
	}

	req.Header.Add("Authorization", "Bearer "+notionApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending search request:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading search response:", err)
		return
	}

	// Parse the search response
	var searchResponse NotionSearchResponse
	err = json.Unmarshal(body, &searchResponse)
	if err != nil {
		fmt.Println("Error unmarshaling search response:", err)
		return
	}

	if len(searchResponse.Results) == 0 {
		fmt.Println("No pages found with the specified name")
		return
	}

	// Archive only the page that matches the exact name
	for _, result := range searchResponse.Results {
		pageID := result.ID

		// Fetch the page details to confirm the exact name match
		pageDetails, err := getPageDetails(pageID)
		if err != nil {
			fmt.Println("Error fetching page details:", err)
			continue
		}

		if pageDetails.Properties.Title.Title[0].Text.Content == pageName {
			archiveURL := "https://api.notion.com/v1/pages/" + pageID

			// Create the request body to archive the page
			bodyData := map[string]bool{
				"archived": true,
			}
			sendData, err := json.Marshal(bodyData)
			if err != nil {
				fmt.Println("Error marshaling archive request:", err)
				continue
			}

			req, err := http.NewRequest("PATCH", archiveURL, bytes.NewBuffer(sendData))
			if err != nil {
				fmt.Println("Error creating archive request:", err)
				continue
			}

			req.Header.Add("Authorization", "Bearer "+notionApiKey)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Notion-Version", "2022-06-28")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Println("Error sending archive request: ", err)
				continue
			}
			defer resp.Body.Close()

			archiveBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Error reading archive response body:", err)
				continue
			}
			fmt.Println("Archived page response:", string(archiveBody))
		}
	}
}

// Helper function to fetch page details by page ID
func getPageDetails(pageID string) (*NotionRequest, error) {
	notionApiKey := GetEnvVar("NOTION_API")
	url := "https://api.notion.com/v1/pages/" + pageID

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+notionApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pageDetails NotionRequest
	err = json.Unmarshal(body, &pageDetails)
	if err != nil {
		return nil, err
	}

	return &pageDetails, nil
}
func sendTextToNotionPage(pageName, pageDescription, pageParagraph string) {
	notionApiKey := GetEnvVar("NOTION_API")
	url := "https://api.notion.com/v1/pages"

	// Split the pageParagraph into chunks of 2000 characters or less
	chunks := splitIntoChunks(pageParagraph, 2000)

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
							Content: pageName,
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
								Content: pageDescription,
							},
							Annotations: struct {
								Bold bool `json:"bold"`
							}{
								Bold: true,
							},
						},
					},
				},
			},
		},
	}

	// Add each chunk as a separate paragraph block
	for _, chunk := range chunks {
		notionRequest.Children = append(notionRequest.Children, Block{
			Object: "block",
			Type:   "paragraph",
			Paragraph: &Paragraph{
				RichText: []RichText{
					{
						Type: "text",
						Text: struct {
							Content string `json:"content"`
						}{
							Content: chunk,
						},
					},
				},
			},
		})
	}

	// Marshal the request body to JSON
	sendData, err := json.Marshal(notionRequest)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(sendData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Add headers
	req.Header.Add("Authorization", "Bearer "+notionApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return
	}
	defer resp.Body.Close()

	// Read and print the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	fmt.Println("Raw response body:", string(body))
}

// Helper function to split a string into chunks of a specified maximum length
func splitIntoChunks(text string, maxLength int) []string {
	var chunks []string
	runes := []rune(text)

	for len(runes) > maxLength {
		chunks = append(chunks, string(runes[:maxLength]))
		runes = runes[maxLength:]
	}

	chunks = append(chunks, string(runes))
	return chunks
}
