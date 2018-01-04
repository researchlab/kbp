package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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
	r := handlers.Router(version.BuildTime, version.Commit, version.Release)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}
	go func() {
		log.Fatal(srv.ListenAndServe())
	}()
	log.Print("The service is ready to listen and serve.")
	killSignal := <-interrupt
	switch killSignal {
	case os.Kill:
		log.Print("Got SIGKILL...")
	case os.Interrupt:
		log.Print("Got SIGINT...")
	case syscall.SIGTERM:
		log.Print("Got SIGTERM...")
	}
	log.Print("The service is shutting down...")
	srv.Shutdown(context.Background())
	log.Print("Done")
}
