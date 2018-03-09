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

// IRequest is a request structure for communicating with ui
type IRequest struct {
	PkgName string `json:"pkgName"`
}

// IResponse is a response structure for communicating with ui
type IResponse struct {
	Graph   *IListGraph `json:"graph"`
	PkgName string      `json:"pkgName"`
}

// IListGraph is a type for the response
type IListGraph struct {
	Nodes []*extractor.Pkg `json:"nodes"`
	Edges []*extractor.Dep `json:"edges"`
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
	http.HandleFunc("/dep", handleGetDeps)
}

func handleGetDeps(writer http.ResponseWriter, request *http.Request) {
	var req IRequest
	err := json.NewDecoder(request.Body).Decode(&req)

	if err != nil {
		http.Error(writer, err.Error(), 400)
		return
	}

	pkgName := req.PkgName
	fmt.Printf("Package name: %s\n", pkgName)

	defer func() {
		if r := recover(); r != nil {
			http.Error(writer, fmt.Sprint(r), 400)
			return
		}
	}()
	nodes, edges, err := extractor.GetDeps(pkgName)

	if err != nil {
		http.Error(writer, err.Error(), 400)
		return
	}

	fmt.Printf("nodes len: %d, edges len: %d\n", len(nodes), len(edges))
	json.NewEncoder(writer).Encode(&IResponse{&IListGraph{nodes, edges}, pkgName})
}
