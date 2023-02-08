package helpers

import (
	"flag"
	"fmt"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

//TraceData trace data struct
type TraceData struct {
	File string
	Line int
	Fn   string
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

//Flags struct
type Flags struct {
	LogLevelVar, ExecutionIdVar, AccountIdVar, OrgIdVar, ProjectIdVar, PipelineIdVar, ApiKeyVar string
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
	flag.Parse()
	return flags
}
