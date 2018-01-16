package main

import (
	"net/http"
	"log"
)

func main() {
	http.Handle("/explorer/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/" {
			http.ServeFile(writer, request, "explorer/index.html")
			return
		}
	})

	if err := http.ListenAndServe("localhost:1111", nil); err != nil {
		log.Fatal(err)
	}
}