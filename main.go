package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/byron1st/godepexplorer/extractor"
)

type reqStruct struct {
	PkgName string `json:"pkgName"`
}

type resStruct struct {
	Nodes []*extractor.Package `json:"nodes"`
	Edges []*extractor.Dep     `json:"edges"`
}

func handlerGenerator(extractorFunc func(string) ([]*extractor.Package, []*extractor.Dep, error)) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var req reqStruct
		err := json.NewDecoder(request.Body).Decode(&req)

		if err != nil {
			http.Error(writer, err.Error(), 400)
		}

		pkgName := req.PkgName
		fmt.Printf("Package name: %s\n", pkgName)
		nodes, edges, err := extractorFunc(pkgName)

		fmt.Printf("nodes len: %d, edges len: %d\n", len(nodes), len(edges))
		json.NewEncoder(writer).Encode(&resStruct{nodes, edges})
	}
}

func main() {
	http.HandleFunc("/dir", handlerGenerator(extractor.GetDirTree))
	http.HandleFunc("/dep", handlerGenerator(extractor.GetDeps))

	if err := http.ListenAndServe("localhost:1111", nil); err != nil {
		log.Fatal(err)
	}
}
