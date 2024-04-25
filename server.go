package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	r := http.NewServeMux()

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), r))
}
