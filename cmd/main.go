package main

import (
	"fmt"
	"golang-api/pkg/http/rest"
	"golang-api/pkg/storage"
	"golang-api/pkg/storage/repository"
	"log"
	"net/http"
)

func main() {
	s, _ := storage.NewStorage()
	repo := repository.NewJobRepository(&s)

	defer s.Close()

	router := rest.Handler(*repo)

	go rest.VisitResult(*repo)
	noOfWorkers := 10
	go rest.CreateWorkerPool(noOfWorkers)

	go rest.PerimeterResult(*repo)
	noOfImageWorkers := 15
	go rest.CreateImageWorkerPool(noOfImageWorkers, *repo)

	fmt.Println("The server is up on : http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
