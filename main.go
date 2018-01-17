package main

import (
	"net/http"
	"log"
	"encoding/json"
	"github.com/byron1st/godepexplorer/extractor"
	"fmt"
)

type dirReqStruct struct {
	RootPkgName string `json:"rootPkgName"`
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

	rootPkgName := req.RootPkgName
	fmt.Printf("Root package name: %s\n", rootPkgName)

	err, nodes, edges := extractor.GetDirTree(rootPkgName)
	fmt.Printf("nodes len: %d, edges len: %d\n", len(nodes), len(edges))
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