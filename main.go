package main

import (
	"net/http"
	"log"
	"encoding/json"
	"github.com/byron1st/godepexplorer/extractor"
	"fmt"
)

type reqStruct struct {
	PkgName string `json:"pkgName"`
}

type resStruct struct {
	Nodes []*extractor.Package `json:"nodes"`
	Edges []*extractor.Dep     `json:"edges"`
}

func handlerGenerator(extractorFunc func(string) (error, []*extractor.Package, []*extractor.Dep)) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var req reqStruct
		err := json.NewDecoder(request.Body).Decode(&req)

		if err != nil {
			http.Error(writer, err.Error(), 400)
		}

		pkgName := req.PkgName
		fmt.Printf("Package name: %s\n", pkgName)
		err, nodes, edges := extractorFunc(pkgName)

		fmt.Printf("nodes len: %d, edges len: %d\n", len(nodes), len(edges))
		json.NewEncoder(writer).Encode(&resStruct{nodes, edges})
	}
}

func main() {
	http.Handle("/explorer/", http.FileServer(http.Dir(".")))

	/**
	POST /dir 해당 dir 내의 서브 dir를 계층구조로 반환. POST data 는 { rootPkgName: "" } 으로 받음.
		response: {
			nodes: [] // dir name list
			edges: [] // composition relation
		}
 	POST /dep
	 */
	http.HandleFunc("/dir", handlerGenerator(extractor.GetDirTree))
	http.HandleFunc("/dep", handlerGenerator(extractor.GetDeps))

	if err := http.ListenAndServe("localhost:1111", nil); err != nil {
		log.Fatal(err)
	}
}