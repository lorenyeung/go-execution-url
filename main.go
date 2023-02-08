package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/lorenyeung/go-execution-url/auth"
	"github.com/lorenyeung/go-execution-url/helpers"

	log "github.com/sirupsen/logrus"
)

var gitCommit string
var version string

type rawData struct {
	Blah          string     `json:"status"`
	DataStructObj DataStruct `json:"data"`
}

type DataStruct struct {
	PipelineExecutionSummaryObj PipelineExecutionSummary `json:"pipelineExecutionSummary"`
	ExecutionGraphObj           ExecutionGraph           `json:"executionGraph"`
}

type PipelineExecutionSummary struct {
	LayoutNodeMapObj map[string]interface{} `json:"layoutNodeMap"`
	StoreType        string                 `json:"storeType"`
}

type LayoutNodeMap struct {
	Name           string `json:"name"`
	NodeIdentifier string `json:"nodeIdentifier"`
	NodeUuid       string `json:"nodeUuid"`
}

type ExecutionGraph struct {
	NodeMapObj map[string]interface{} `json:"nodeMap"`
}

type NodeMap struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	Uuid       string `json:"uuid"`
	BaseFqn    string `json:"baseFqn"`
	Status     string `json:"status"`
}

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

	url := "https://app.harness.io/gateway/pipeline/api/pipelines/execution/v2/" + flags.ExecutionIdVar + "?accountIdentifier=" + flags.AccountIdVar + "&orgIdentifier=" + flags.OrgIdVar + "&projectIdentifier=" + flags.ProjectIdVar + "&renderFullBottomGraph=true"
	m := map[string]string{
		"x-api-key": flags.ApiKeyVar,
	}
	data, respcode, _ := auth.GetRestAPI("GET", false, url, "", "", "", m, 0)
	if respcode != 200 {
		log.Panic("error")
	} else {
		var executionData rawData
		err := json.Unmarshal(data, &executionData)
		if err != nil {
			log.Panic(err)
		}

		for key, value := range executionData.DataStructObj.ExecutionGraphObj.NodeMapObj {
			stepBody, _ := json.Marshal(value)
			var NodeMapObj NodeMap
			err = json.Unmarshal(stepBody, &NodeMapObj)
			if err != nil {
				log.Panic(err)
			}

			for key2, value2 := range executionData.DataStructObj.PipelineExecutionSummaryObj.LayoutNodeMapObj {

				//skip rollback stages
				if !strings.Contains(key2, "_rollbackStage") {
					stageBody, _ := json.Marshal(value2)
					var LayoutNodeMapObj LayoutNodeMap
					err = json.Unmarshal(stageBody, &LayoutNodeMapObj)
					if err != nil {
						log.Panic(err)
					}

					if strings.Contains(NodeMapObj.BaseFqn, "pipeline.stages."+LayoutNodeMapObj.NodeIdentifier) {
						fmt.Println(LayoutNodeMapObj.Name, "|", NodeMapObj.Name, "|", NodeMapObj.Status, "|", LayoutNodeMapObj.NodeIdentifier, "|", NodeMapObj.Identifier)
						fmt.Println("https://app.harness.io/ng/#/account/" + flags.AccountIdVar + "/ci/orgs/" + flags.OrgIdVar + "/projects/" + flags.ProjectIdVar + "/pipelines/" + flags.PipelineIdVar + "/executions/" + flags.ExecutionIdVar + "/pipeline?storeType=" + executionData.DataStructObj.PipelineExecutionSummaryObj.StoreType + "&stage=" + key2 + "&step=" + key + "&childStage=&stageExecId=")
						break
					}
				}
			}
		}
	}
}
