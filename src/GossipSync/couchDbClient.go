package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var COUCHDB_HOST string = "172.18.0.184"

type CouchObjectDocument struct {
	Function string
	Host     string
	Port     string
}

type CouchReturnDocument struct {
	OK  string `json:"ok"`
	Id  string `json:"id"`
	Rev string `json:"rev"`
}

type CouchResultDocument struct {
	Docs     []CouchObjectDocument `json:"docs"`
	Bookmark string                `json:"bookmark"`
}

func SaveNode(functionName string, host string, port string) CouchReturnDocument {
	data := CouchObjectDocument{
		Function: functionName,
		Host:     host,
		Port:     port,
	}
	dataJson, _ := json.Marshal(data)
	resp, err := http.Post(fmt.Sprintf("http://openwhisk:openwhisk@%s:5984/status_sync/", COUCHDB_HOST), "application/json;charset=utf-8", bytes.NewBuffer(dataJson))
	if err != nil {
		fmt.Println("无法将节点数据保存到 CouchDB: ", err.Error())
	}
	defer resp.Body.Close()

	var info CouchReturnDocument
	json.NewDecoder(resp.Body).Decode(&info)
	return info
}

func GetNodes(functionName string) []CouchObjectDocument {
	selector := fmt.Sprintf("{\"selector\": {\"Function\": \"%s\"}, \"fields\": [\"Function\", \"Host\", \"Port\"]}", functionName)
	resp, err1 := http.Post(fmt.Sprintf("http://openwhisk:openwhisk@%s:5984/status_sync/_find", COUCHDB_HOST), "application/json;charset=utf-8", strings.NewReader(selector))
	if err1 != nil {
		fmt.Println("无法查询运行", functionName, "函数的节点", err1.Error())
		return []CouchObjectDocument{}
	}
	defer resp.Body.Close()

	var result CouchResultDocument
	json.NewDecoder(resp.Body).Decode(&result)
	// fmt.Println(result)
	return result.Docs
}

func DeleteNode(info CouchReturnDocument) {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	del, _ := http.NewRequest("DELETE", fmt.Sprintf("http://openwhisk:openwhisk@%s:5984/status_sync/%s?rev=%s", COUCHDB_HOST, info.Id, info.Rev), nil)
	_, err := httpClient.Do(del)
	if err != nil {
		fmt.Println("无法删除节点信息:", err.Error())
	}
}
