package main

import (
	"encoding/json"
	"fmt"
	"time"
)

func main() {
	/*reader := bufio.NewReader(os.Stdin)
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
	}

	fmt.Print("Type Y to send to notion: ")
	resp, _ := reader.ReadString('\n')
	resp = strings.TrimSpace(resp)
	fmt.Println("entered: " + resp)
	course := "Geology"

	todos := []string{}
	dt := time.Now()
	if resp == "Y" {
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
				if dt.Before(dueAtTime.Add(168*time.Hour)) && !discussion.Locked_For_User {
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
				if dt.Before(dueAtTime.Add(168*time.Hour)) && !discussion.Assignment.Locked_For_User {
					fmt.Print("executed\n")
					todo = "Assignment: " + discussion.Title + " Due at: " + todo
					todos = append(todos, todo)
				}

			}
		}
	}

	SendToNotion(course+" Assignments as of "+FormatDate(dt), todos)
	SendAllAssignmentsToNotion()*/
	//DeleteNotionPage("cdf832e3-454f-47cf-ab04-d2d63d4a6e00")
	//GetAllAssignmentsByModule()
	//GetAllAssignments()

	//Main call
	ArchivePageByName(FormatDate(time.Now()) + " Assignments and Discussions Due Within a Month")
	ArchivePageByName(FormatDate(time.Now().AddDate(0, 0, -1)) + " Assignments and Discussions Due Within a Month")
	notionRequest := SendAllAssignmentsToOneNotionPage()
	chatgptData, err := json.Marshal(notionRequest)
	if err != nil {
		fmt.Println("error marshalling chatpgt json")
	}

	if time.Now().Weekday() == time.Monday {
		response := generateWeeklySchedule(string(chatgptData))
		fmt.Println(response)
		sendTextToNotionPage(FormatDate(time.Now())+" ChatGPT Weekly Schedule", "ChatGPT generated weekly schedule", response)
	}

	//End of main call
	//updateToDoList("cdf832e3-454f-47cf-ab04-d2d63d4a6e00", todos)
}
func FormatDate(t time.Time) string {
	return t.Format("01/02/2006")
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
