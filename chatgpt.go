package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type ChatGPTRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatGPTResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
		Index        int     `json:"index"`
	} `json:"choices"`
}

func chatGptQuery(notionData string, partialQuery string) string {
	apiKey := GetEnvVar("CHATGPT_KEY")
	url := "https://api.openai.com/v1/chat/completions"

	// Set up the request body with partialQuery
	requestBody := ChatGPTRequest{
		Model: "gpt-4",
		Messages: []Message{
			{
				Role: "system",
				Content: `You are a helpful assistant and will read the data and extract the assignments for me and list them by date. My schedule is as follows:
				- **Mondays**: Electronics 1 class at 9:00 AM to 10:15 AM, Computer communication networks class from 12:00 PM to 1:15 PM, Electronics lab every other week (I went August 26th 2024, so I do not have to go September 2nd) from 3:00 PM to 5:50 PM, and Cinema survey class from 6:00 PM to 8:50 PM.
				- **Tuesdays**: Research meeting with Justin at 11:00 AM, Operating systems class at 3:00 PM to 4:20 PM.
				- **Wednesdays**: Electronics 1 class at 9:00 AM to 10:15 AM, Computer communication networks class from 12:00 PM to 1:15 PM, Coffee shop visit with girlfriend Sam at 2:00 PM to 3:00 PM.
				- **Thursdays**: Operating systems class at 3:00 PM to 4:20 PM.
				- **Fridays, Saturdays, Sundays**: No classes.

				Weekly obligations:
				- Gym 4 to 5 days a week, typically for 1 to 1 ½ hours.
				- Coffee shop visit every Wednesday with Sam at about 2:00 PM to 3:00 PM arrival, 7:00 PM to 8:00 PM leaving.
				- 2 hours of studying 5-6 days a week for interview preparation.
				- 5 to 10 hours of work each week.
				- I also want one completely free day except on busy weeks where I may have one or more exams.

				Please create a nicely formatted daily schedule for every single day of the next week, incorporating my assignments and weekly obligations.  
				In the schedule, make sure to include a time to wake up at, I like to wake up at 9:30 am on days I have no obligations.
				Also include the time at which I will complete my obligations and an estimated end time.
				Do not forget to leave time for eating, as well as a 1 hour wind down routine at the end of each night.
				I typically like to go to bed between 1 and 2 am but don't forget to include my nightly routine.
				Make sure each daily task has a time started and ended associated with it in the format 3:00 pm to 4:00 pm.
				Make sure to include potential time driving and extra study time during the day.
				Assume it takes me 30 minutes to get ready and 20 minutes to drive to and from school.
				Please create a nicely formatted daily schedule for every single day of the next week, incorporating my assignments and weekly obligations. 
				Make the format such that:
				**Day of the week plus Date**
				-Task
				-Task
				replace task with each thing I have to do that day and the time.
				Also add a fun history fact at the end of each day as well as a daily motivating quote and daily protestant christian bible verse.
				Base your calendar off of the date September 2nd 2024 being a Monday as well and don't forget my quote, fact, and bible verse.`,
			},
			{
				Role:    "user",
				Content: notionData + partialQuery, // Include the partial query to continue from the last response
			},
		},
	}

	// Marshal the request body to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println("Error marshaling request body:", err)
		return ""
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return ""
	}

	// Add headers
	req.Header.Add("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return ""
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return ""
	}

	// Print the raw response body (for debugging)
	fmt.Println("Raw response body:", string(body))

	// Parse the response
	var response ChatGPTResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Error unmarshaling response:", err)
		return ""
	}

	// Return the formatted message
	if len(response.Choices) > 0 {
		return response.Choices[0].Message.Content
	}

	return ""
}
func generateWeeklySchedule(notionData string) string {
	var fullSchedule strings.Builder
	var lastResponse string
	daysCovered := make(map[string]bool)

	for {
		// Call the chatGptQuery with the last part of the response (if any)
		lastResponse = chatGptQuery(notionData, lastResponse)
		if lastResponse == "" {
			break
		}
		fullSchedule.WriteString(lastResponse)

		// Update daysCovered based on the latest response
		updateDaysCovered(lastResponse, daysCovered)

		// Check if all days of the week have been covered
		if len(daysCovered) == 7 {
			break
		}
	}

	return formatSchedule(fullSchedule.String())
}

// Update the daysCovered map based on the presence of each day in the response
func updateDaysCovered(response string, daysCovered map[string]bool) {
	daysOfWeek := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	for _, day := range daysOfWeek {
		if strings.Contains(response, day) {
			daysCovered[day] = true
		}
	}
}

// Helper function to format the final schedule nicely
func formatSchedule(schedule string) string {
	return strings.TrimSpace(schedule)
}
