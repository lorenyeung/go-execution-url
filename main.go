package main

import (
	"container/list"
	"flag"
	"fmt"
	"os"

	"github.com/lorenyeung/go-execution-url/auth"
	"github.com/lorenyeung/go-execution-url/helpers"

	log "github.com/sirupsen/logrus"
)

var gitCommit string
var version string

func printVersion() {
	fmt.Println("Current build version:", gitCommit, "Current Version:", version)
}

func main() {
	versionFlag := flag.Bool("v", false, "Print the current version and exit")
	flags := helpers.SetFlags()
	helpers.SetLogger(flags.LogLevelVar)

	switch {
	case *versionFlag:
		printVersion()
		return
	}

	if flags.AccountIdVar == "" || flags.ApiKeyVar == "" || flags.ExecutionIdVar == "" || flags.OrgIdVar == "" || flags.PipelineIdVar == "" || flags.ProjectIdVar == "" {
		log.Panic("please set all required flags")
	}
	SortedData := list.New()
	longestName := auth.GetData(flags, SortedData)
	outfileData := helpers.PrintData(longestName, flags, SortedData)
	if flags.OutfileVar != "" {
		err := os.WriteFile(flags.OutfileVar, outfileData, 0644)
		if err != nil {
			log.Error(err)
		}
	}
}
