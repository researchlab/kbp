package main

import (
	"log"
	"net/http"
	"os"

	"github.com/researchlab/advent-2017/handlers"
	"github.com/researchlab/advent-2017/version"
)

func main() {
	log.Printf("Starting the service...\ncommit: %s, build time: %s, release: %s",
		version.Commit, version.BuildTime, version.Release)
	port := os.Getenv("PORT")
	if len(port) == 0 {
		log.Fatal("Port is not set.")
	}
	router := handlers.Router(version.BuildTime, version.Commit, version.Release)
	log.Print("The service is ready to listen and serve.")
	log.Fatal(http.ListenAndServe(":"+port, router))
}
