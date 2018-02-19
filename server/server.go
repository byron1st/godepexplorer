package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/byron1st/godepexplorer/extractor"
)

// Server struct
type Server struct {
	host string
	port int
}

type reqStruct struct {
	PkgName string `json:"pkgName"`
}

type resStruct struct {
	Nodes []*extractor.Package `json:"nodes"`
	Edges []*extractor.Dep     `json:"edges"`
}

type errResStruct struct {
	Error string `json:"error"`
}

// MakeServer creates and returns a new Server object.
func MakeServer(host string, port int) *Server {
	setRoute()

	server := &Server{host, port}
	// TODO: check the validity of host and port.
	return server
}

// StartServer starts the corresponding server.
func (server *Server) StartServer() {
	if err := http.ListenAndServe(fmt.Sprintf("%s:%d", server.host, server.port), nil); err != nil {
		log.Fatal(err)
	}
}

func setRoute() {
	http.HandleFunc("/dir", handlerGenerator(extractor.GetDirTree))
	http.HandleFunc("/dep", handlerGenerator(extractor.GetDeps))
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

		defer func() {
			if r := recover(); r != nil {
				json.NewEncoder(writer).Encode(&errResStruct{fmt.Sprint(r)})
			}
		}()
		nodes, edges, err := extractorFunc(pkgName)

		fmt.Printf("nodes len: %d, edges len: %d\n", len(nodes), len(edges))
		json.NewEncoder(writer).Encode(&resStruct{nodes, edges})
	}
}
