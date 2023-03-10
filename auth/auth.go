package auth

import (
	"bytes"
	"container/list"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lorenyeung/go-execution-url/helpers"

	log "github.com/sirupsen/logrus"
)

//GetRestAPI GET rest APIs response with error handling
func GetRestAPI(method string, auth bool, urlInput, userName, apiKey, providedfilepath string, header map[string]string, retry int) ([]byte, int, http.Header) {
	if retry > 5 {
		log.Warn("Exceeded retry limit, cancelling further attempts")
		return nil, 0, nil
	}

	body := new(bytes.Buffer)
	//PUT upload file
	if method == "PUT" && providedfilepath != "" {
		//req.Header.Set()
		file, err := os.Open(providedfilepath)
		helpers.Check(err, false, "open", helpers.Trace())
		defer file.Close()

		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", filepath.Base(providedfilepath))
		helpers.Check(err, false, "create", helpers.Trace())
		io.Copy(part, file)
		err = writer.Close()
		helpers.Check(err, false, "writer close", helpers.Trace())
	}

	client := http.Client{}
	req, err := http.NewRequest(method, urlInput, body)
	if auth {
		req.SetBasicAuth(userName, apiKey)
	}
	for x, y := range header {
		log.Debug("Recieved extra header:", x+":"+y)
		req.Header.Set(x, y)
	}

	if err != nil {
		log.Warn("The HTTP request failed with error", err)
	} else {

		resp, err := client.Do(req)
		helpers.Check(err, false, "The HTTP response", helpers.Trace())

		if err != nil {
			return nil, 0, nil
		}
		// need to account for 403s with xray, or other 403s, 429? 204 is bad too (no content for docker)
		switch resp.StatusCode {
		case 200:
			log.Debug("Received ", resp.StatusCode, " OK on ", method, " request for ", urlInput, " continuing")
		case 201:
			if method == "PUT" {
				log.Debug("Received ", resp.StatusCode, " ", method, " request for ", urlInput, " continuing")
			}
		case 403:
			log.Error("Received ", resp.StatusCode, " Forbidden on ", method, " request for ", urlInput, " continuing")
			// should we try retry here? probably not
		case 404:
			log.Debug("Received ", resp.StatusCode, " Not Found on ", method, " request for ", urlInput, " continuing")
		case 429:
			log.Error("Received ", resp.StatusCode, " Too Many Requests on ", method, " request for ", urlInput, ", sleeping then retrying, attempt ", retry)
			time.Sleep(10 * time.Second)
			GetRestAPI(method, auth, urlInput, userName, apiKey, providedfilepath, header, retry+1)
		case 204:
			if method == "GET" {
				log.Error("Received ", resp.StatusCode, " No Content on ", method, " request for ", urlInput, ", sleeping then retrying")
				time.Sleep(10 * time.Second)
				GetRestAPI(method, auth, urlInput, userName, apiKey, providedfilepath, header, retry+1)
			} else {
				log.Debug("Received ", resp.StatusCode, " OK on ", method, " request for ", urlInput, " continuing")
			}
		case 500:
			log.Error("Received ", resp.StatusCode, " Internal Server error on ", method, " request for ", urlInput, " failing out")
			return nil, 0, nil
		default:
			log.Warn("Received ", resp.StatusCode, " on ", method, " request for ", urlInput, " continuing")
		}
		//Mostly for HEAD requests
		statusCode := resp.StatusCode
		headers := resp.Header

		if providedfilepath != "" && method == "GET" {
			// Create the file
			out, err := os.Create(providedfilepath)
			helpers.Check(err, false, "File create:"+providedfilepath, helpers.Trace())
			defer out.Close()

			//done := make(chan int64)
			//go helpers.PrintDownloadPercent(done, filepath, int64(resp.ContentLength))
			_, err = io.Copy(out, resp.Body)
			helpers.Check(err, false, "The file copy:"+providedfilepath, helpers.Trace())
		} else {
			//maybe skip the download or retry if error here, like EOF
			data, err := ioutil.ReadAll(resp.Body)
			helpers.Check(err, false, "Data read:"+urlInput, helpers.Trace())
			if err != nil {
				log.Warn("Data Read on ", urlInput, " failed with:", err, ", sleeping then retrying, attempt:", retry)
				time.Sleep(10 * time.Second)

				GetRestAPI(method, auth, urlInput, userName, apiKey, providedfilepath, header, retry+1)
			}

			return data, statusCode, headers
		}
	}
	return nil, 0, nil
}

func GetData(flags helpers.Flags, SortedData *list.List) (longestName int) {
	url := "https://app.harness.io/gateway/pipeline/api/pipelines/execution/v2/" + flags.ExecutionIdVar + "?accountIdentifier=" + flags.AccountIdVar + "&orgIdentifier=" + flags.OrgIdVar + "&projectIdentifier=" + flags.ProjectIdVar + "&renderFullBottomGraph=true"
	m := map[string]string{
		"x-api-key": flags.ApiKeyVar,
	}
	longestName = 0
	data, respcode, _ := GetRestAPI("GET", false, url, "", "", "", m, 0)
	if respcode != 200 {
		log.Panic("error")
	} else {
		var executionData helpers.RawData
		err := json.Unmarshal(data, &executionData)
		if err != nil {
			log.Panic(err)
		}

		for key, value := range executionData.DataStructObj.ExecutionGraphObj.NodeMapObj {
			stepBody, _ := json.Marshal(value)
			var NodeMapObj helpers.NodeMap
			err = json.Unmarshal(stepBody, &NodeMapObj)
			if err != nil {
				log.Panic(err)
			}

			for key2, value2 := range executionData.DataStructObj.PipelineExecutionSummaryObj.LayoutNodeMapObj {

				//skip rollback stages
				if !strings.Contains(key2, "_rollbackStage") {
					stageBody, _ := json.Marshal(value2)
					var LayoutNodeMapObj helpers.LayoutNodeMap
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
								log.Debug("update longestName:", longestName)
							}
							object := helpers.DataArray{NodeMapObj, LayoutNodeMapObj, finalurl}
							if SortedData.Len() == 0 {
								SortedData.PushFront(object)
								log.Debug("first date push", object.NodeMapObj.EndTs)
							} else {
								//sort data and insert according to epoch of end timestamp
								for e := SortedData.Front(); e != nil; e = e.Next() {
									v := e.Value.(helpers.DataArray)
									if object.NodeMapObj.EndTs < v.NodeMapObj.EndTs {
										SortedData.InsertBefore(object, e)
										log.Debug("insert before")
										break
									} else {
										if e == SortedData.Back() {
											SortedData.PushBack(object)
											log.Debug("insert after")
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
	}
	log.Debug("Final max step length:", longestName)
	return longestName
}
