package main

import (
	"container/list"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/lorenyeung/go-execution-url/auth"
	"github.com/lorenyeung/go-execution-url/helpers"
	"github.com/savioxavier/termlink"

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
	EndTs      int    `json:"endTs"`
}

type DataArray struct {
	NodeMapObj       NodeMap
	LayoutNodeMapObj LayoutNodeMap
	FinalURL         string
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

		//arrayData := []DataArray{}
		// sort by time
		SortedData := list.New()
		longestName := 0

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
						log.Debug("unsorted object:", LayoutNodeMapObj.Name, "|", NodeMapObj.Name, "|", NodeMapObj.Status, "|", LayoutNodeMapObj.NodeIdentifier, "|", NodeMapObj.Identifier, NodeMapObj.EndTs)
						//TODO childstage and stageExecId
						finalurl := "https://app.harness.io/ng/#/account/" + flags.AccountIdVar + "/ci/orgs/" + flags.OrgIdVar + "/projects/" + flags.ProjectIdVar + "/pipelines/" + flags.PipelineIdVar + "/executions/" + flags.ExecutionIdVar + "/pipeline?storeType=" + executionData.DataStructObj.PipelineExecutionSummaryObj.StoreType + "&stage=" + key2 + "&step=" + key + "&childStage=&stageExecId="
						if NodeMapObj.Name != "Execution" {
							if len(NodeMapObj.Name) > longestName {
								longestName = len(NodeMapObj.Name)
							}
							object := DataArray{NodeMapObj, LayoutNodeMapObj, finalurl}
							if SortedData.Len() == 0 {
								SortedData.PushFront(object)
								log.Debug("first date push", object.NodeMapObj.EndTs)
							} else {
								//sort data and insert according to epoch of end timestamp
								for e := SortedData.Front(); e != nil; e = e.Next() {
									v := e.Value.(DataArray)
									if object.NodeMapObj.EndTs < v.NodeMapObj.EndTs {
										SortedData.InsertBefore(object, e)
										break
									} else {
										if e == SortedData.Back() {
											SortedData.PushBack(object)
											break
										}
									}
								}
							}
						}
						break
					}
				}
			}
		}

		//print data
		count := 0
		line := []string{strings.Repeat("-", longestName+38)}
		printStage := true
		fmt.Printf("%3s | %-*.*s | %-*.*s | %6s\n", "No.", longestName, longestName, "Step Name", 13, 13, "End time", "Execution URL")
		for e := SortedData.Front(); e != nil; e = e.Next() {
			v := e.Value.(DataArray)

			if v.NodeMapObj.Name == v.LayoutNodeMapObj.Name {
				printStage = true
			} else {
				if printStage {

					fmt.Println(strings.Join(line, ""), "\nStage:", v.LayoutNodeMapObj.Name)
					fmt.Println(strings.Join(line, ""))
					printStage = false
				}
				fmt.Printf("%3d | %-*.*s | %13d | %6s\n", count, longestName, longestName, v.NodeMapObj.Name, v.NodeMapObj.EndTs, termlink.Link("Execution", v.FinalURL))
				count++
			}
		}
	}
}
