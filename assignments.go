package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type assignment_due struct {
	Due_At                    string `json:"due_at"`
	Name                      string `json:"name"`
	Id                        int    `json:"id"`
	Has_Submitted_Submissions bool   `json:"has_submitted_submissions"`
	Is_Quiz_Assignment        bool   `json:"is_quiz_assignment"`
	Require_Lockdown_Browser  bool   `json:"require_lockdown_browser"`
	Locked_For_User           bool   `json:locked_for_user`
}

type discussion_due struct {
	Due_At      string         `json:"due_at"`
	Title       string         `json:"title"`
	Id          int            `json:"id"`
	Description string         `json:"description"`
	Assignment  assignment_due `json:"assignment"`
	Created_At  string         `json:created_at"`
}

func GetAllAssignments() []assignment_due {
	canvasApiKey := GetEnvVar("CANVAS_API")
	url := "https://webcourses.ucf.edu/api/v1/users/4374518/courses/1464091/assignments?fields=id,due_at,points_possible,name,submission_types"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("error creating request:", err)
		return []assignment_due{}
	}

	req.Header.Add("Authorization", "Bearer "+canvasApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return []assignment_due{}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return []assignment_due{}
	}

	var prettyJSON1 bytes.Buffer
	error := json.Indent(&prettyJSON1, body, "", "\t")
	if error != nil {
		log.Println("JSON parse error: ", error)

		return []assignment_due{}
	}

	f, err := os.OpenFile("./text_response.txt", os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}

	n3, err := f.WriteString(prettyJSON1.String())
	check(err)
	fmt.Printf("wrote %d bytes\n", n3)

	var result []assignment_due
	json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("Error unmarshaling JSON: ", err)
		return []assignment_due{}
	}
	fmt.Printf("Found %d assignments\n", len(result))
	fmt.Println("Success from GetAll")
	for _, assignment := range result {
		fmt.Println("name: " + assignment.Name)
		fmt.Println("due at: " + assignment.Due_At)
	}
	resultsJSON, _ := json.Marshal(result)
	var prettyJSON2 bytes.Buffer
	error2 := json.Indent(&prettyJSON2, resultsJSON, "", "\t")
	if error2 != nil {
		log.Println("JSON parse error: ", error)

		return []assignment_due{}
	}

	f2, err := os.OpenFile("./os_results.txt", os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}

	n2, err := f2.WriteString(prettyJSON2.String())
	check(err)
	fmt.Printf("wrote %d bytes\n", n2)
	//fmt.Println(string(body))
	return result
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func GetDiscussionPost() []discussion_due {
	canvasApiKey := GetEnvVar("CANVAS_API")
	url := "https://webcourses.ucf.edu/api/v1/courses/1461901/discussion_topics"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("error creating request:", err)
		return []discussion_due{}
	}

	req.Header.Add("Authorization", "Bearer "+canvasApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return []discussion_due{}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return []discussion_due{}
	}
	//fmt.Println("Raw response body:", string(body))
	var prettyJSON1 bytes.Buffer
	error := json.Indent(&prettyJSON1, body, "", "\t")
	if error != nil {
		log.Println("JSON parse error: ", error)

		return []discussion_due{}
	}

	f, err := os.OpenFile("./discussion_response.txt", os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}

	n3, err := f.WriteString(prettyJSON1.String())
	check(err)
	fmt.Printf("wrote %d bytes\n", n3)

	var result []discussion_due
	json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("Error unmarshaling JSON: ", err)
		return []discussion_due{}
	}
	fmt.Printf("Found %d discussions\n", len(result))
	fmt.Println("Success from GetAll")
	for _, discussion := range result {
		fmt.Println("name: " + discussion.Title)
		fmt.Println("due at: " + discussion.Assignment.Due_At)
	}
	resultsJSON, _ := json.Marshal(result)
	var prettyJSON2 bytes.Buffer
	error2 := json.Indent(&prettyJSON2, resultsJSON, "", "\t")
	if error2 != nil {
		log.Println("JSON parse error: ", error)

		return []discussion_due{}
	}

	f2, err := os.OpenFile("./disucssion_results.txt", os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}

	n2, err := f2.WriteString(prettyJSON2.String())
	check(err)
	fmt.Printf("wrote %d bytes\n", n2)
	//fmt.Println(string(body))
	return result
}

func GetAllAssignmentsByCourse(course int) []assignment_due {
	canvasApiKey := GetEnvVar("CANVAS_API")
	url := fmt.Sprintf("https://webcourses.ucf.edu/api/v1/users/4374518/courses/%d/assignments", course)
	fmt.Println(url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("error creating request:", err)
		return []assignment_due{}
	}

	req.Header.Add("Authorization", "Bearer "+canvasApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return []assignment_due{}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return []assignment_due{}
	}

	var prettyJSON1 bytes.Buffer
	error := json.Indent(&prettyJSON1, body, "", "\t")
	if error != nil {
		log.Println("JSON parse error: ", error)

		return []assignment_due{}
	}

	f, err := os.OpenFile("./text_response.txt", os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}

	n3, err := f.WriteString(prettyJSON1.String())
	check(err)
	fmt.Printf("wrote %d bytes\n", n3)

	var result []assignment_due
	json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("Error unmarshaling JSON: ", err)
		return []assignment_due{}
	}
	fmt.Printf("Found %d assignments\n", len(result))
	fmt.Println("Success from GetAll")
	for _, assignment := range result {
		fmt.Println("name: " + assignment.Name)
		fmt.Println("due at: " + assignment.Due_At)
	}
	resultsJSON, _ := json.Marshal(result)
	var prettyJSON2 bytes.Buffer
	error2 := json.Indent(&prettyJSON2, resultsJSON, "", "\t")
	if error2 != nil {
		log.Println("JSON parse error: ", error)

		return []assignment_due{}
	}

	f2, err := os.OpenFile("./os_results.txt", os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}

	n2, err := f2.WriteString(prettyJSON2.String())
	check(err)
	fmt.Printf("wrote %d bytes\n", n2)
	//fmt.Println(string(body))
	return result
}

func GetDiscussionPostByCourse(course int) []discussion_due {
	canvasApiKey := GetEnvVar("CANVAS_API")
	url := fmt.Sprintf("https://webcourses.ucf.edu/api/v1/courses/%d/discussion_topics", course)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("error creating request:", err)
		return []discussion_due{}
	}

	req.Header.Add("Authorization", "Bearer "+canvasApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return []discussion_due{}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return []discussion_due{}
	}
	//fmt.Println("Raw response body:", string(body))
	var prettyJSON1 bytes.Buffer
	error := json.Indent(&prettyJSON1, body, "", "\t")
	if error != nil {
		log.Println("JSON parse error: ", error)

		return []discussion_due{}
	}

	f, err := os.OpenFile("./discussion_response.txt", os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}

	n3, err := f.WriteString(prettyJSON1.String())
	check(err)
	fmt.Printf("wrote %d bytes\n", n3)

	var result []discussion_due
	json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("Error unmarshaling JSON: ", err)
		return []discussion_due{}
	}
	fmt.Printf("Found %d discussions\n", len(result))
	fmt.Println("Success from GetAll")
	for _, discussion := range result {
		fmt.Println("name: " + discussion.Title)
		fmt.Println("due at: " + discussion.Assignment.Due_At)
	}
	resultsJSON, _ := json.Marshal(result)
	var prettyJSON2 bytes.Buffer
	error2 := json.Indent(&prettyJSON2, resultsJSON, "", "\t")
	if error2 != nil {
		log.Println("JSON parse error: ", error)

		return []discussion_due{}
	}

	f2, err := os.OpenFile("./disuccsion_results.txt", os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}

	n2, err := f2.WriteString(prettyJSON2.String())
	check(err)
	fmt.Printf("wrote %d bytes\n", n2)
	//fmt.Println(string(body))
	return result
}
