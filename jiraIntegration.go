package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
)

type JiraConfig struct {
	Url      string
	Username string
	Token    string
}

type JiraIssue struct {
	Key    string          `json:"key"`
	Fields JiraIssueFields `json:"fields"`
}

type JiraIssueFields struct {
	Summary      string                `json:"summary"`
	Description  string                `json:"description"`
	Created      string                `json:"created"`
	TimeTracking JiraIssueTimeTracking `json:"timetracking"`
}

type JiraIssueTimeTracking struct {
	OriginalEstimateSeconds  int `json:"originalEstimateSeconds"`
	RemainingEstimateSeconds int `json:"remainingEstimateSeconds"`
	TimeSpentSeconds         int `json:"timeSpentSeconds"`
}

var jiraHttpClient *http.Client
var jiraCurrentTask JiraIssue

func _checkAndLoadJiraIssue(taskKey string) bool {
	if _configIsComplete() {
		if jiraHttpClient == nil {
			jiraHttpClient = &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					auth := base64.StdEncoding.EncodeToString([]byte(config.JiraServiceConfig.Username + ":" + config.JiraServiceConfig.Token))
					req.Header.Add("Authorization", "Basic "+auth)

					return nil
				},
			}
		}

		req, err := http.NewRequest("GET", config.JiraServiceConfig.Url+"/issue/"+taskKey, nil)
		check(err)

		auth := base64.StdEncoding.EncodeToString([]byte(config.JiraServiceConfig.Username + ":" + config.JiraServiceConfig.Token))
		req.Header.Add("Authorization", "Basic "+auth)

		response, err := jiraHttpClient.Do(req)
		check(err)

		defer response.Body.Close()

		if response.StatusCode == 200 {
			json.NewDecoder(response.Body).Decode(&jiraCurrentTask)
		} else if response.StatusCode == 401 {
			fmt.Println("Warning: Unable to update Jira, please check your configuration.")
		}

		return response.StatusCode == 200
	} else {
		fmt.Println("Warning: Unable to update Jira, please check your configuration.")

		return false
	}
}

func _submitJiraWorkLog(taskKey string, info TaskDescription, seconds int64) bool {
	if _configIsComplete() {
		if jiraHttpClient == nil {
			jiraHttpClient = &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					auth := base64.StdEncoding.EncodeToString([]byte(config.JiraServiceConfig.Username + ":" + config.JiraServiceConfig.Token))
					req.Header.Add("Authorization", "Basic "+auth)

					return nil
				},
			}
		}

		var summary string
		if config.billable_enable {
			summary = fmt.Sprintf("Job Type: %s\nStatus: %s\nDescription: %s", info.JobType, info.Status, info.Description)
		} else {
			summary = fmt.Sprintf("Job Type: %s\nDescription: %s", info.JobType, info.Description)
		}

		body, err := json.Marshal(map[string]interface{}{
			"comment":          summary,
			"timeSpentSeconds": seconds,
		})
		check(err)

		req, err := http.NewRequest("POST", config.JiraServiceConfig.Url+"/issue/"+taskKey+"/worklog", bytes.NewBuffer(body))
		req.Header.Set("Content-type", "application/json")
		check(err)

		auth := base64.StdEncoding.EncodeToString([]byte(config.JiraServiceConfig.Username + ":" + config.JiraServiceConfig.Token))
		req.Header.Add("Authorization", "Basic "+auth)

		response, err := jiraHttpClient.Do(req)
		check(err)

		defer response.Body.Close()

		return response.StatusCode == 201
	}

	return false
}

func isJiraTaskFormat(identifier string) bool {
	taskFmt := regexp.MustCompile(`(?i)[A-Z0-9]+-[0-9]+`)

	return taskFmt.Match([]byte(identifier))
}
