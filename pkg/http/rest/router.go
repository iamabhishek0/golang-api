package rest

import (
	"github.com/julienschmidt/httprouter"
	"golang-api/pkg/storage/repository"
	"net/http"
)

func Handler(job repository.JobStorage) http.Handler {
	router := httprouter.New()

	router.POST("/api/submit", addJob(job))
	router.GET("/api/status", getJobStatus(job))

	return router
}
