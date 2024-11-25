package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type GitlabConfig struct {
	Url            string
	Token          string
	DefaultProject string
}

type GitlabUser struct {
	Id       int32  `json:"id"`
	Username string `json:"username"`
}

type GitlabProject struct {
	Id          int32  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

type GitlabIssue struct {
	Id    int32  `json:"id"`
	Iid   int32  `json:"iid"`
	State string `json:"state"`
	Title string `json:"title"`
}

func isGitlabTaskFormat(identifier string) bool {
	taskFmt := regexp.MustCompile(`(?i)[0-9]+-[A-Z0-9]+`)

	return taskFmt.Match([]byte(identifier))
}

var gitlabHttpClient = &http.Client{}
var gitlabUser = GitlabUser{}
var gitlabProject = GitlabProject{}
var gitlabIssue = GitlabIssue{}

func loadGitlabUser() bool {
	response, err := gitlabApiRequest("GET", "/user")
	check(err)
	defer response.Body.Close()

	if response.StatusCode == 200 {
		json.NewDecoder(response.Body).Decode(&gitlabUser)

		return true
	}

	fmt.Println("Warning: Unable to update gitlab, please check your configuration.")

	return false
}

func loadGitlabProject(id string) bool {
	response, err := gitlabApiRequest("GET", "/projects/"+id)
	check(err)
	defer response.Body.Close()

	if response.StatusCode == 200 {
		json.NewDecoder(response.Body).Decode(&gitlabProject)

		return true
	}

	return false
}

func checkAndLoadGitlabIssue(issueKey string) bool {
	if _configIsComplete() {
		var didLoadProject = false

		if config.GitlabServiceConfig.DefaultProject != "" {
			didLoadProject = loadGitlabProject(config.GitlabServiceConfig.DefaultProject)

			if didLoadProject {
				issueKeyParts := strings.Split(issueKey, "-")
				response, err := gitlabApiRequest("GET", fmt.Sprintf("/projects/%d/issues/%s", gitlabProject.Id, issueKeyParts[0]))
				check(err)
				defer response.Body.Close()

				if response.StatusCode == 200 {
					json.NewDecoder(response.Body).Decode(&gitlabIssue)

					return true
				}
			}

			return false
		} else {
			// @TODO: search for the current project via name
		}
	}

	return false
}

func submitGitlabTimeSpent(info TaskDescription, seconds int64) bool {
	if gitlabProject.Id != 0 && gitlabIssue.Iid != 0 { // project and iid are always non-zero
		var summary string

		if config.billable_enable {
			summary = fmt.Sprintf("Job Type: %s\nStatus: %s\nDescription: %s", info.JobType, info.Status, info.Description)
		} else {
			summary = fmt.Sprintf("Job Type: %s\nDescription: %s", info.JobType, info.Description)
		}

		response, err := gitlabApiRequest("POST", fmt.Sprintf("/projects/%d/issues/%d/add_spent_time?duration=%ds&summary=%s", gitlabProject.Id, gitlabIssue.Iid, seconds, url.QueryEscape(summary)))
		check(err)

		if response.StatusCode == 201 {
			return true
		}

		fmt.Println("Warning: Unable to update gitlab, please check your configuration.")

		return false
	}

	return false
}

func gitlabApiRequest(method string, path string) (*http.Response, error) {
	req, err := http.NewRequest(method, config.GitlabServiceConfig.Url+"/api/v4"+path, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("PRIVATE-TOKEN", config.GitlabServiceConfig.Token)

	return gitlabHttpClient.Do(req)
}
