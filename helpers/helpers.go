package helpers

import (
	"container/list"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/savioxavier/termlink"
	log "github.com/sirupsen/logrus"
)

//TraceData trace data struct
type TraceData struct {
	File string
	Line int
	Fn   string
}

type RawData struct {
	Status        string     `json:"status"`
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

//Flags struct
type Flags struct {
	LogLevelVar, ExecutionIdVar, AccountIdVar, OrgIdVar, ProjectIdVar, PipelineIdVar, ApiKeyVar, OutputVar, OutfileVar string
	ForceLinkVar, ShowIdsVar                                                                                           bool
}

//Trace get function data
func Trace() TraceData {
	var trace TraceData
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Warn("Failed to get function data")
		return trace
	}

	fn := runtime.FuncForPC(pc)
	trace.File = file
	trace.Line = line
	trace.Fn = fn.Name()
	return trace
}

//SetLogger sets logger settings
func SetLogger(logLevelVar string) {
	level, err := log.ParseLevel(logLevelVar)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)

	log.SetReportCaller(true)
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.QuoteEmptyFields = true
	customFormatter.FullTimestamp = true
	customFormatter.CallerPrettyfier = func(f *runtime.Frame) (string, string) {
		repopath := strings.Split(f.File, "/")
		function := strings.Replace(f.Function, "go-pkgdl/", "", -1)
		return fmt.Sprintf("%s\t", function), fmt.Sprintf(" %s:%d\t", repopath[len(repopath)-1], f.Line)
	}

	log.SetFormatter(customFormatter)
	log.Info("Log level set at ", level)
}

//Check logger for errors
func Check(e error, panicCheck bool, logs string, trace TraceData) {
	if e != nil && panicCheck {
		log.Error(logs, " failed with error:", e, " ", trace.Fn, " on line:", trace.Line)
		panic(e)
	}
	if e != nil && !panicCheck {
		log.Warn(logs, " failed with error:", e, " ", trace.Fn, " on line:", trace.Line)
	}
}

func PrintData(longestName int, flags Flags, SortedData *list.List) []byte {
	count := 0
	statuslen := 12
	timelen := 13
	if flags.ShowIdsVar {
		longestName = longestName*2 + 7
	}
	totallen := longestName + statuslen + timelen + 28
	line := []string{strings.Repeat("-", totallen)}
	printStage := true
	var outfileTable string
	if flags.OutputVar == "table" {

		header := fmt.Sprintf("%3s | %-*.*s | %-*.*s | %-*.*s | %6s\n", "No.", longestName, longestName, "Step", statuslen, statuslen, "Status", timelen, timelen, "End time", "Execution URL")
		io.WriteString(os.Stdout, header)
		outfileTable = header
		for e := SortedData.Front(); e != nil; e = e.Next() {
			v := e.Value.(DataArray)

			if v.NodeMapObj.Name == v.LayoutNodeMapObj.Name {
				printStage = true
			} else {
				if printStage {
					stageId := ""
					if flags.ShowIdsVar {
						stageId = "(id:" + v.LayoutNodeMapObj.NodeIdentifier + ")"
					}
					stage := fmt.Sprintf("%*s\nStage: %-s %-s\n%*s\n", totallen, strings.Join(line, ""), v.LayoutNodeMapObj.Name, stageId, totallen, strings.Join(line, ""))
					io.WriteString(os.Stdout, stage)
					outfileTable = outfileTable + stage
					printStage = false
				}
				stepId := ""
				if flags.ShowIdsVar {
					stepId = "(id:" + v.NodeMapObj.Identifier + ")"
				}
				tableData := fmt.Sprintf("%3d | %-*.*s | %-*.*s | %*.*d | %6s\n", count, longestName, longestName, v.NodeMapObj.Name+" "+stepId, statuslen, statuslen, v.NodeMapObj.Status, timelen, timelen, v.NodeMapObj.EndTs, termlink.Link("Execution", v.FinalURL, flags.ForceLinkVar))
				io.WriteString(os.Stdout, tableData)
				outfileTable = outfileTable + tableData
				count++
			}
		}
		return []byte(outfileTable)
	}
	if flags.OutputVar == "json" {
		jsonDataRaw := []DataArray{}
		for e := SortedData.Front(); e != nil; e = e.Next() {
			v := e.Value.(DataArray)
			if v.NodeMapObj.Name == v.LayoutNodeMapObj.Name {
			} else {
				jsonDataRaw = append(jsonDataRaw, v)
			}
		}
		jsondata, err := json.Marshal(jsonDataRaw)
		if err != nil {
			log.Error(err)
		} else {
			fmt.Println(string(jsondata))
			return jsondata
		}
	}
	return []byte(outfileTable)
}

//SetFlags function
func SetFlags() Flags {
	var flags Flags
	flag.StringVar(&flags.LogLevelVar, "log", "INFO", "Order of Severity: TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC")
	flag.StringVar(&flags.ExecutionIdVar, "exe", "", "Execution ID")
	flag.StringVar(&flags.AccountIdVar, "acc", "", "Account ID")
	flag.StringVar(&flags.OrgIdVar, "org", "default", "Organisation ID")
	flag.StringVar(&flags.ProjectIdVar, "pro", "default", "Project ID")
	flag.StringVar(&flags.PipelineIdVar, "pip", "", "Pipeline ID")
	flag.StringVar(&flags.ApiKeyVar, "key", "", "API Key")
	flag.StringVar(&flags.OutputVar, "output", "table", "Output format: table, json")
	flag.StringVar(&flags.OutfileVar, "outfile", "", "optionally write results to a file")
	flag.BoolVar(&flags.ForceLinkVar, "forcelink", false, "Force link print")
	flag.BoolVar(&flags.ShowIdsVar, "showid", false, "Show Harness IDs for steps and stages")
	flag.Parse()
	return flags
}
