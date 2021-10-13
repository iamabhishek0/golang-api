package rest

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/xid"
	"golang-api/pkg/storage/repository"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Job struct {
	StoreId   string   `json:"store_id"`
	ImageUrl  []string `json:"image_url"`
	VisitTime string   `json:"visit_time"`
}

type allJob struct {
	Count  int   `json:"count"`
	Visits []Job `json:"visits"`
}

type StoreIdError struct {
	StoreID string `json:"store_id"`
	Error   string `json:"error"`
}

type FailedStoreID struct {
	Status string         `json:"status"`
	JobId  string         `json:"job_id"`
	Error  []StoreIdError `json:"error"`
}

type WorkerJob struct {
	jobid string
	data  Job
	job   repository.JobStorage
}
type Result struct {
	jobid  string
	status *int
}

type ImageJob struct {
	URL     string
	jobid   string
	storeid string
	job     repository.JobStorage
	status  *int
}

type ImageResult struct {
	perimeter int
	jobid     string
	storeid   string
}

var visitJobs = make(chan WorkerJob, 10)
var results = make(chan Result, 10)
var imageJobs = make(chan ImageJob, 15)
var imageResults = make(chan ImageResult, 15)

func genXid() string {
	id := xid.New()
	return id.String()
}

// Handling each visit
func allocateVisits(data []Job, jobid string, job repository.JobStorage) {
	for i := 0; i < len(data); i++ {
		visitJobs <- WorkerJob{jobid: jobid, data: data[i], job: job}
	}
}

func worker(wg *sync.WaitGroup) {
	for job := range visitJobs {
		var status = 1
		processURL(job.jobid, job.data, &status, job.job)
		results <- Result{job.jobid, &status}
	}
	wg.Done()
}

func createWorkerPool(noOfWorkers int) {
	var wg sync.WaitGroup
	for i := 0; i < noOfWorkers; i++ {
		wg.Add(1)
		go worker(&wg)
	}
	wg.Wait()
	close(results)
}


func result(job repository.JobStorage) {
	for result := range results {
		repository.UpdateJobStatus(result.jobid, *result.status, job)
	}
}

// Handling each image
func imageWorker(wg *sync.WaitGroup, jobDB repository.JobStorage) {
	for job := range imageJobs {
		perimeter, err := processPerimeter(job)
		if err == nil {
			imageResults <- ImageResult{perimeter, job.jobid, job.storeid}
		}
		if *job.status==2 {
			repository.UpdateJobStatus(job.jobid, *job.status, jobDB)
		}
	}
	wg.Done()
}

func createImageWorkerPool(noOfWorkers int, job repository.JobStorage) {
	var wg sync.WaitGroup
	for i := 0; i < noOfWorkers; i++ {
		wg.Add(1)
		go imageWorker(&wg, job)
	}
	wg.Wait()
	close(imageResults)
}

func allocateImage(data []string, jobid string, storeid string, job repository.JobStorage, status *int) {
	for i := 0; i < len(data); i++ {
		imageJobs <- ImageJob{URL: data[i], jobid: jobid, storeid: storeid, job: job, status: status}
	}
}

func perimeterResult(job repository.JobStorage) {
	for result := range imageResults {
		_, _ = repository.AddImage(result.jobid, result.storeid, result.perimeter, job)
	}
}

func processPerimeter(data ImageJob) (perimeter int, err error) {
	path := data.URL
	resp, err := http.Get(path)
	if err != nil {
		*data.status = 2
		_, err := repository.AddFailed(data.jobid, data.storeid, data.job)
		return 0, err
	}
	defer resp.Body.Close()
	m, _, err := image.Decode(resp.Body)

	if err != nil {
		*data.status = 2
		_, _ = repository.AddFailed(data.jobid, data.storeid, data.job)
		return 0, err
	}

	g := m.Bounds()
	height := g.Dy()
	width := g.Dx()
	perimeter = 2 * (height + width)

	rand.Seed(time.Now().Unix())
	randomNum := 10000 + rand.Intn(400-100)
	time.Sleep(time.Duration(randomNum) * time.Millisecond)

	return perimeter, nil
}

func processURL(jobid string, data Job, status *int, job repository.JobStorage) {
	go allocateImage(data.ImageUrl, jobid, data.StoreId, job, status)
	go perimeterResult(job)

	noOfWorkers := 15
	go createImageWorkerPool(noOfWorkers, job)
}

func addJob(job repository.JobStorage) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		decoder := json.NewDecoder(r.Body)
		var newJob allJob
		err := decoder.Decode(&newJob)
		if err != nil || len(newJob.Visits) != newJob.Count {
			w.WriteHeader(400)
			m := make(map[string]string)
			if err != nil {
				m["error"] = "Missing parameters"
			} else {
				m["error"] = "Count of visits is incorrect"
			}
			json.NewEncoder(w).Encode(m)
			return
		}

		for i := 0; i < len(newJob.Visits); i++ {
			if len(newJob.Visits[i].StoreId) == 0 || len(newJob.Visits[i].ImageUrl) == 0 || len(newJob.Visits[i].ImageUrl) == 0 {
				w.WriteHeader(400)
				m := make(map[string]string)
				m["error"] = "Missing parameters"
				json.NewEncoder(w).Encode(m)
				return
			}
		}
		// get a unique ID of 20 chars
		jobid := genXid()
		_, err = repository.AddStatus(jobid, job)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		go allocateVisits(newJob.Visits, jobid, job)

		noOfWorkers := 10
		go createWorkerPool(noOfWorkers)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		m := make(map[string]string)
		m["jobid"] = jobid
		json.NewEncoder(w).Encode(m)
	}
}

func getJobStatus(job repository.JobStorage) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		jobid, ok := r.URL.Query()["jobid"]

		if !ok || len(jobid[0]) < 1 {
			w.WriteHeader(400)
			m := make(map[string]string)
			m["error"] = "Incorrect Parameter"
			json.NewEncoder(w).Encode(m)
			return
		}

		key := jobid[0]
		key = string(key)
		status, err := repository.CheckStatus(key, job)

		if err != nil {
			w.WriteHeader(400)
			m := make(map[string]string)
			m["error"] = "jobid does not exist"
			json.NewEncoder(w).Encode(m)
			return
		}

		if status == 0 || status == 1 {
			w.WriteHeader(200)
			m := make(map[string]string)
			if status == 0 {
				m["status"] = "ongoing"
			} else {
				m["status"] = "completed"
			}
			m["job_id"] = key
			json.NewEncoder(w).Encode(m)
			return
		} else {
			w.WriteHeader(400)
			storelist, _ := repository.GetFailedStoreId(key, job)

			m := FailedStoreID{JobId: key, Status: "failed"}
			for _, id := range storelist {
				storeID := StoreIdError{Error: ""}
				storeID.StoreID = id
				m.Error = append(m.Error, storeID)
			}
			json.NewEncoder(w).Encode(m)
			return
		}
	}
}
