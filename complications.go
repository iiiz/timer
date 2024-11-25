package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func ps1Complication() {
	if _statusFileExists() {
		task, startTimeString := _readStatusFile()
		startTime, err := time.Parse(time.RFC3339, startTimeString)
		check(err)

		formattedDiff := _formatDuration(time.Since(startTime))

		fmt.Print(fmt.Sprintf("%s %s", task, formattedDiff))
	} else {
		fmt.Print("<No task>")
	}
}

func preCmd() {
	cwd, err := os.Getwd()
	check(err)
	pwd, last_wd_err := _getLastWorkingDir()

	if !_statusFileExists() && last_wd_err == nil {
		// no current status, check the cwd and see if it is a git dir.
		isGit := _isGitRepo()

		if isGit && cwd != pwd {
			head := _getHeadRef()
			branchLeader := strings.Split(head, "/")[2]
			var isPossibleTaskIdent = false

			if config.upstream_service != "" {
				switch config.upstream_service {
				case "jira":
					isPossibleTaskIdent = isJiraTaskFormat(branchLeader)
				case "gitlab":
					isPossibleTaskIdent = isGitlabTaskFormat(branchLeader)
				}
			}

			reassign := !isPossibleTaskIdent
			reader := bufio.NewReader(os.Stdin)
			var answer string = ""
			var taskIdent string = ""

			if isPossibleTaskIdent {
				fmt.Print(fmt.Sprintf("timer: Git repository detected, would you like to start tracking time with %s as the task id?\n (y/n/o): ", branchLeader))
			} else {
				fmt.Print("timer: Git repository detected, would you like to start tracking time?\n (y/n): ")
			}
			answer, _ = reader.ReadString('\n')
			answer = strings.Replace(answer, "\n", "", -1)

			switch answer {
			case "y":
				if !reassign {
					taskIdent = branchLeader
				}
			case "n":
				_setWorkingDir(cwd)
				os.Exit(0)
			case "o":
				reassign = true
			default:
				_setWorkingDir(cwd)
				fmt.Println("Unknown response, exiting.")
				os.Exit(0)
			}

			if reassign {
				fmt.Print("Enter a task identifier: ")
				newIdent, err := reader.ReadString('\n')
				check(err)

				taskIdent = strings.Replace(newIdent, "\n", "", -1)
			}

			start(taskIdent, "")
		}
	}

	_setWorkingDir(cwd)
}
