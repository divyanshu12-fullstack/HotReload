package main

import (
	"fmt"
	"log"
	"net/http"
)

const version = "v2"

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from testserver %s!\n", version)
	})
	log.Println("testserver listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
