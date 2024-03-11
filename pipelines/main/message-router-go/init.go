package main

import (
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	scalebox "github.com/kaichao/scalebox/golang/misc"
	"github.com/sirupsen/logrus"
)

var (
	logger *logrus.Logger

	hosts = []string{"10.11.16.79", "10.11.16.76", "10.11.16.75"}
	// hosts            = []string{"10.11.16.79", "10.11.16.80", "10.11.16.76", "10.11.16.75"}
	// numNodesPerGroup int

	localMode bool

	workDir string
)

func init() {
	var err error

	workDir = os.Getenv("WORD_DIR")
	if workDir == "" {
		workDir = "/work"
	}

	logger = logrus.New()
	level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		level = logrus.WarnLevel
	}
	logger.SetLevel(level)
	logger.SetReportCaller(true)

	localMode = os.Getenv("LOCAL_MODE") == "yes"
}

func sendNodeAwareMessage(message string, headers map[string]string, sinkJob string, num int) int {
	if !localMode {
		scalebox.AppendToFile("/work/messages.txt", sinkJob+","+message)
		return 0
	}

	toHost := hosts[num%len(hosts)]
	cmdTxt := fmt.Sprintf("scalebox task add --upsert --sink-job %s --to-ip %s %s", sinkJob, toHost, message)
	if len(headers) > 0 {
		h, err := json.Marshal(headers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "headers:%v,JSON marshaling failed:%v\n", headers, err)
		} else {
			cmdTxt = fmt.Sprintf("scalebox task add --upsert --sink-job %s --to-ip %s --headers '%s' %s", sinkJob, toHost, h, message)
		}
	}

	fmt.Printf("cmd-text:%s\n", cmdTxt)
	code, stdout, stderr := scalebox.ExecShellCommandWithExitCode(cmdTxt, 10)
	fmt.Printf("stdout for task-add:\n%s\n", stdout)
	fmt.Fprintf(os.Stderr, "stderr for task-add:\n%s\n", stderr)
	return code
}
