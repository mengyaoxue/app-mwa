package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	scalebox "github.com/kaichao/scalebox/golang/misc"
	"github.com/sirupsen/logrus"
)

func createSemaphore(semaName string, defaultValue int) int {
	cmdText := fmt.Sprintf("scalebox semaphore create %s %d", semaName, defaultValue)
	code, stdout, stderr := scalebox.ExecShellCommandWithExitCode(cmdText, 15)
	fmt.Printf("stdout for task-add:\n%s\n", stdout)
	fmt.Fprintf(os.Stderr, "stderr for task-add:\n%s\n", stderr)
	return code
}

func countDown(semaName string) int {
	cmdText := fmt.Sprintf("scalebox semaphore countdown %s", semaName)
	code, stdout, stderr := scalebox.ExecShellCommandWithExitCode(cmdText, 15)
	fmt.Printf("exit-code for semaphore countdown:\n%d\n", code)
	fmt.Printf("stdout for semaphore countdown:\n%s\n", stdout)
	fmt.Fprintf(os.Stderr, "stderr for semaphore countdown:\n%s\n", stderr)
	if code > 0 {
		return -10
	}
	code, err := strconv.Atoi(strings.TrimSpace(stdout))
	if err != nil {
		fmt.Fprintf(os.Stderr, "stderr for convert to code in semaphore countdown:\n%v\n", err)
		return -11
	}

	return code
}

func getSemaphore(semaName string) int {
	cmdText := fmt.Sprintf("scalebox semaphore get %s", semaName)
	code, stdout, stderr := scalebox.ExecShellCommandWithExitCode(cmdText, 15)
	fmt.Printf("exit-code for semaphore get:\n%d\n", code)
	fmt.Printf("stdout for semaphore get:\n%s\n", stdout)
	fmt.Fprintf(os.Stderr, "stderr for semaphore get:\n%s\n", stderr)
	if code > 0 {
		return -10
	}
	code, err := strconv.Atoi(strings.TrimSpace(stdout))
	if err != nil {
		fmt.Fprintf(os.Stderr, "stderr for convert to code in semaphore get:\n%v\n", err)
		return -11
	}

	return code
}

// Sema ...
type Sema struct {
	name  string
	value int
}

func doInsert(values []Sema) {
	if !batchInsert {
		for _, sema := range values {
			createSemaphore(sema.name, sema.value)
		}
		return
	}
	// start transaction
	tx, err := db.Begin()
	if err != nil {
		logrus.Errorf("err:%v\n", err)
	}
	defer tx.Rollback()

	jobID, _ := strconv.Atoi(os.Getenv("JOB_ID"))
	sqlText := `
		INSERT INTO t_semaphore(name,value,value0,app)
		SELECT $1,$2,$2,app FROM t_job WHERE id=$3
		ON CONFLICT (name,app) DO UPDATE SET (value,value0) = ($2,$2)
	`

	batchSize := 100
	for i := 0; i < len(values); i += batchSize {
		stmt, err := tx.Prepare(sqlText)
		if err != nil {
			logrus.Errorf("err:%v\n", err)
		}
		defer stmt.Close()

		end := i + batchSize
		if end > len(values) {
			end = len(values)
		}

		for _, v := range values[i:end] {
			if _, err := stmt.Exec(v.name, v.value, jobID); err != nil {
				logrus.Errorf("err:%v\n", err)
			}
		}
		if err = tx.Commit(); err != nil {
			logrus.Errorf("err:%v\n", err)
		}

		fmt.Printf("[%d..%d], %d row(s) inserted.\n", i, end, end-i)

		// start next batch
		if tx, err = db.Begin(); err != nil {
			logrus.Errorf("err:%v\n", err)
		}
	}
}

var (
	// used for semaphore batch-insert
	db          *sql.DB
	batchInsert bool
)

func init() {
	// localMode = os.Getenv("LOCAL_MODE") == "yes"
	batchInsert = os.Getenv("BATCH_INSERT") == "yes"

	dbHost := os.Getenv("PGHOST")
	if dbHost == "" {
		dbHost = scalebox.GetLocalIP()
	}
	dbPort := os.Getenv("PGPORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	databaseURL := fmt.Sprintf("postgres://scalebox:changeme@%s:%s/scalebox", dbHost, dbPort)
	// set database connection
	var err error
	if db, err = sql.Open("pgx", databaseURL); err != nil {
		log.Fatal("Unable to connect to database:", err)
	}
	db.SetConnMaxLifetime(500)
	db.SetMaxIdleConns(50)
	db.SetMaxOpenConns(20)
	db.Stats()
}
