package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("What would you like to do?")
	fmt.Print("Type 1 to get Assignments: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	fmt.Println("entered: " + name)
	//canvasApiKey := GetEnvVar("CANVAS_API")
	var assignments []assignment_due
	var discussions []discussion_due
	if name == "1" {
		assignments = GetAllAssignments()
		discussions = GetDiscussionPost()
	} else {
		assignments = []assignment_due{}
		discussions = []discussion_due{}
	}

	/*if len(assignments) == 0 || len(discussions) == 0 {
		fmt.Println("Assignments or discussions  is empty")
		return
	} else {
		fmt.Println("Assignments is not empty")
	}*/

	fmt.Print("Type Y to send to notion: ")
	resp, _ := reader.ReadString('\n')
	resp = strings.TrimSpace(resp)
	fmt.Println("entered: " + resp)
	course := "Geology"

	todos := []string{}
	if resp == "Y" {
		if len(assignments) != 0 {

		} else {
			for _, discussion := range assignments {
				todo := formatTime(discussion.Due_At)
				todo = "Assignment: " + discussion.Name + "Due at: " + todo
				todos = append(todos, todo)
			}
		}
		if len(discussions) != 0 {
			for _, discussion := range discussions {

				todo := formatTime(discussion.Assignment.Due_At)
				todo = "Assignment: " + discussion.Title + "Due at: " + todo
				todos = append(todos, todo)
			}
		} else {
		}
	}
	SendToNotion(course, todos)
}

func formatTime(Time string) string {
	t, err := time.Parse(time.RFC3339, Time)
	if err != nil {
		fmt.Println("Error parsing time:", err)
		return ""
	}

	// Convert UTC to Eastern Time (EST or EDT depending on the date)
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		fmt.Println("Error loading location:", err)
		return ""
	}
	t = t.In(loc)

	// Format the time in the desired format
	formattedTime := t.Format("01/02/2006 @ 03:04PM MST")
	return formattedTime
}
