package main

import (
	"fmt"
	"strings"

	"github.com/k8sbykeshed/k8s-service-validator/tools"
	"github.com/savaki/jq"
	"go.uber.org/zap"
)

type Workflow struct {
	name               string
	status             string
	id                 string
	lastRunID          string
	lastJobID          string
	lastJobCompletedAt string
	passedTests        map[string]bool
	failedTests        map[string]bool
}

var wfmap map[string]*Workflow

const (
	cmd          = "/usr/local/bin/gh"
	keyPassInLog = "PASS tests"
	keyFailInLog = "FAIL: tests"
)

func main() {
	// TODO gh auth login
	wfmap = make(map[string]*Workflow)
	data, err := tools.RunCmd(cmd, "workflow", "list")
	if err != nil {
		zap.L().Fatal(err.Error())
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		ss := strings.Split(line, "\t")
		wf := &Workflow{name: strings.TrimSpace(ss[0]), status: strings.TrimSpace(ss[1]), id: strings.TrimSpace(ss[2])}
		wfmap[wf.id] = wf
	}

	for id, wf := range wfmap {
		wf.passedTests = make(map[string]bool)
		wf.failedTests = make(map[string]bool)
		data, err := tools.RunCmd(cmd, "run", "list", "--workflow", id, "--limit", "1")
		if err != nil {
			fmt.Println(err.Error())
		}
		ss := strings.Split(string(data), "\t")
		runID := ss[len(ss)-3]
		wf.lastRunID = runID

		// get job id from run id
		data, err = tools.RunCmd(cmd, "api", fmt.Sprintf(
			"https://api.github.com/repos/K8sbykeshed/k8s-service-validator/actions/runs/%v/jobs", runID))
		if err != nil {
			fmt.Printf("error to fetch runner's job: %v \n", err)
		}

		// parse json response
		opJobID, err := jq.Parse(".jobs.[0].id")
		if err != nil {
			fmt.Printf("error to fetch runner's job id: %v \n", err)
		}
		lastJobID, err := opJobID.Apply(data)
		if err != nil {
			fmt.Printf("error to parse response: %v \n", err)
		}

		opCompletedTime, err := jq.Parse(".jobs.[0].completed_at")
		if err != nil {
			fmt.Printf("error to fetch job's completed time: %v \n", err)
		}
		completedTime, err := opCompletedTime.Apply(data)
		if err != nil {
			fmt.Printf("error to parse response: %v \n", err)
		}

		wf.lastJobID = string(lastJobID)
		wf.lastJobCompletedAt = string(completedTime)

		// retrieve and process job log
		data, err = tools.RunCmd(cmd, "run", "view", "--job", wf.lastJobID, "--log")
		if err != nil {
			fmt.Printf("error to fetch job logs: %v \n", err)
		}
		logLines := strings.Split(string(data), "\n")
		for _, line := range logLines {
			if index := strings.Index(line, keyPassInLog); index != -1 {
				l := processLine(line, index)
				if l != "" {
					wf.passedTests[l] = true
				}
			} else if index := strings.Index(line, keyFailInLog); index != -1 {
				l := processLine(line, index)
				if l != "" {
					wf.failedTests[l] = true
				}
			}
		}
	}

	// report
	// currently only in std, can be extendable to more endpoints
	report := generateReport()
	fmt.Println(report)
}

func processLine(l string, index int) string {
	if len(l) > index+11 {
		l = l[index+11:]
		ll := strings.Split(l, "/")
		if len(ll) == 2 {
			return ll[1]
		}
	}
	return ""
}

func generateReport() string {
	header := "Aggregated testing report from CIs running with various CNIs and proxies:"
	lines := []string{header}
	for _, wf := range wfmap {
		lines = append(lines, "\n======================")
		wfResult := "FAILED"
		if len(wf.failedTests) == 0 {
			wfResult = "PASSED"
		}
		lines = append(lines, fmt.Sprintf("[ %v ] - %v", wfResult, wf.name),
			fmt.Sprintf("* status: %s, action run ID: %s, completed at: %v",
				wf.status, wf.lastRunID, wf.lastJobCompletedAt))

		if len(wf.passedTests) != 0 {
			lines = append(lines, "PASSED TESTS:")
			for t := range wf.passedTests {
				lines = append(lines, "- "+t)
			}
		} else {
			lines = append(lines, "PASSED TESTS: NONE")
		}

		if len(wf.failedTests) != 0 {
			lines = append(lines, "FAILED TESTS:")
			for t := range wf.failedTests {
				lines = append(lines, "- "+t)
			}
		} else {
			lines = append(lines, "FAILED TESTS: NONE")
		}
	}
	return strings.Join(lines, "\n")
}
