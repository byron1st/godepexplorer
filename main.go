package main

import (
	"net/http"
	"log"
	"encoding/json"
	"github.com/byron1st/godepexplorer/extractor"
)

type dirReqStruct struct {
	RootPath string
}

type dirResStruct struct {
	Nodes []*extractor.Package `json:"nodes"`
	Edges []*extractor.Dep     `json:"edges"`
}

func getDir(writer http.ResponseWriter, request *http.Request) {
	var req dirReqStruct
	err := json.NewDecoder(request.Body).Decode(&req)

	if err != nil {
		http.Error(writer, err.Error(), 400)
	}

	err, nodes, edges := extractor.GetDirTree(req.RootPath)
	json.NewEncoder(writer).Encode(&dirResStruct{nodes, edges})
}

func main() {
	http.Handle("/explorer/", http.FileServer(http.Dir(".")))

	/**
	GET /dir 해당 dir 내의 서브 dir를 계층구조로 반환
		response: {
			nodes: [] // dir name list
			edges: [] // composition relation
		}
 	GET /dep?nodeId={}
	 */
	http.HandleFunc("/dir", getDir)

	if err := http.ListenAndServe("localhost:1111", nil); err != nil {
		log.Fatal(err)
	}
}