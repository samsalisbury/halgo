package main

import (
	"github.com/samsalisbury/halgo"
	"net/http"
)

func main() {
	if server, err := halgo.NewServer(RootResource{}); err != nil {
		halgo.Print(err)
	} else {
		halgo.Print("Listening on :8080")
		http.ListenAndServe(":8080", server)
	}
}
