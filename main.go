package main

import "github.com/byron1st/godepexplorer/server"

func main() {
	s := server.MakeServer("localhost", 1111)
	s.StartServer()
}
