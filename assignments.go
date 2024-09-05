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

type external_tools struct {
}

type module_assignment struct {
	Content_Id int    `json:"content_id"` //the field needed to search for the assignment.due_ate
	Title      string `json:"title"`
	Type       string `json:"type"`
	Id         int    `json:"id"`
}

type module_info struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Unlock_At string `json:"unlock_at"`
	State     string `json:"state"`
}

func GetAllAssignments() []assignment_due {
	canvasApiKey := GetEnvVar("CANVAS_API")
	//url := "https://webcourses.ucf.edu/api/v1/courses/1461901/assignments"
	//url := "https://webcourses.ucf.edu/api/v1/courses/1461901/assignments/8554277"
	url := "https://webcourses.ucf.edu/api/v1/courses/1465496/assignments"

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

/*
GetExternalTools() []external_tools {

}
*/
func GetAssignmentById(id int) assignment_due {
	canvasApiKey := GetEnvVar("CANVAS_API")
	url := fmt.Sprintf("https://webcourses.ucf.edu/api/v1/courses/1461901/assignments/%d", id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return assignment_due{}
	}

	req.Header.Add("Authorization", "Bearer "+canvasApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return assignment_due{}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return assignment_due{}
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: received status code %d for assignment ID %d\n", resp.StatusCode, id)
		return assignment_due{}
	}

	// Unmarshal into a single assignment_due object
	var result assignment_due
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("Error unmarshaling JSON for assignment ID:", id, err)
		return assignment_due{}
	}

	if result.Name == "" {
		fmt.Printf("Warning: Empty assignment retrieved for ID %d\n", id)
	}

	return result
}

func GetModuleAssignments(module_id int) []module_assignment {
	canvasApiKey := GetEnvVar("CANVAS_API")
	url := fmt.Sprintf("https://webcourses.ucf.edu/api/v1/courses/1461901/modules/%d/items", module_id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("error creating request:", err)
		return nil
	}

	req.Header.Add("Authorization", "Bearer "+canvasApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil
	}

	// Pretty print the JSON
	var prettyJSON1 bytes.Buffer
	if err := json.Indent(&prettyJSON1, body, "", "\t"); err != nil {
		log.Println("JSON parse error: ", err)
		return nil
	}

	// Save pretty JSON to file
	if err := ioutil.WriteFile("./text_response.txt", prettyJSON1.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}

	// Unmarshal into a slice of module_assignment
	var result []module_assignment
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("Error unmarshaling JSON: ", err)
		return nil
	}

	// Print the pretty JSON to the console as a string
	fmt.Println(prettyJSON1.String())

	// Save the final result as JSON to another file
	resultsJSON, _ := json.Marshal(result)
	var prettyJSON2 bytes.Buffer
	if err := json.Indent(&prettyJSON2, resultsJSON, "", "\t"); err != nil {
		log.Println("JSON parse error: ", err)
		return nil
	}

	if err := ioutil.WriteFile("./os_results.txt", prettyJSON2.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}

	return result
}

func GetModules() []module_info {
	canvasApiKey := GetEnvVar("CANVAS_API")
	url := "https://webcourses.ucf.edu/api/v1/courses/1461901/modules"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("error creating request:", err)
		return []module_info{}
	}

	req.Header.Add("Authorization", "Bearer "+canvasApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return []module_info{}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return []module_info{}
	}

	var prettyJSON1 bytes.Buffer
	error := json.Indent(&prettyJSON1, body, "", "\t")
	if error != nil {
		log.Println("JSON parse error: ", error)

		return []module_info{}
	}

	f, err := os.OpenFile("./text_response.txt", os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}

	n3, err := f.WriteString(prettyJSON1.String())
	check(err)
	fmt.Printf("wrote %d bytes\n", n3)

	var result []module_info
	json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("Error unmarshaling JSON: ", err)
		return []module_info{}
	}
	fmt.Printf("Found %d assignments\n", len(result))
	fmt.Println("Success from GetAll")
	/*for _, assignment := range result {
		fmt.Println("name: " + assignment.Name)
		fmt.Println("module id: " + string(assignment.Id))
	}*/
	resultsJSON, _ := json.Marshal(result)
	var prettyJSON2 bytes.Buffer
	error2 := json.Indent(&prettyJSON2, resultsJSON, "", "\t")
	if error2 != nil {
		log.Println("JSON parse error: ", error)

		return []module_info{}
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

func GetAllAssignmentsByModule() []assignment_due {
	Modules := GetModules()
	var ModuleAssignments []module_assignment
	//fmt.Printf("Modules: %d\n", len(Modules))

	for _, module := range Modules {
		moduleItems := GetModuleAssignments(module.Id) // Get all items in the module
		for _, item := range moduleItems {
			//fmt.Println("ModuleAssignment.Type = " + item.Type)
			//fmt.Println("ModuleAssignment.Title = " + item.Title)
			if item.Type == "Assignment" || item.Type == "Quiz" {
				ModuleAssignments = append(ModuleAssignments, item)
			}
		}
	}

	// You can now use ModuleAssignments as needed, e.g., print them or further process them
	fmt.Println("Module Assignments:", ModuleAssignments)
	fmt.Println("\n\n\n\n")
	var assignmentsArr []assignment_due

	for _, assignment := range ModuleAssignments {
		assignments := GetAssignmentById(assignment.Content_Id)

		assignmentsArr = append(assignmentsArr, assignments)

	}

	return assignmentsArr
}

func GetAllAssignmentsByCourse(course int) []assignment_due {
	canvasApiKey := GetEnvVar("CANVAS_API")
	url := fmt.Sprintf("https://webcourses.ucf.edu/api/v1/users/4374518/courses/%d/assignments", course)
	//fmt.Println(url)
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
