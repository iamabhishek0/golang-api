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
	"log"
	"math/rand"
	"net/http"
	"time"
)

type Job struct {
	StoreId    string   `json:"store_id"`
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

func genXid() string{
	id := xid.New()
	return id.String()
}

func processImage(job repository.JobStorage, data []Job, jobid string) error{
	var status = 1
	for i:=0; i < len(data); i++ {
		storeid := data[i].StoreId

		for j:=0; j < len(data[i].ImageUrl); j++ {
			path:=data[i].ImageUrl[j]

			resp, err := http.Get(path)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()
			m, _, err := image.Decode(resp.Body)
			if err != nil {
				status = 2
				_, err := repository.AddFailed(jobid, storeid, job)
				if err != nil {
					return err
				}
				break;
			}
			g := m.Bounds()
			height := g.Dy()
			width := g.Dx()
			perimeter := 2 * (height + width)

			rand.Seed(time.Now().Unix())
			randomNum := 100 + rand.Intn(400-100)
			time.Sleep(time.Duration(randomNum) * time.Millisecond)

			_, _ = repository.AddImage(jobid, storeid, perimeter, job)
			if err != nil {
				return err
			}
		}
	}

	_, err := repository.UpdateJobStatus(jobid, status, job)
	if err != nil {
		return err
	}
	return nil
}

func addJob(job repository.JobStorage) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		decoder := json.NewDecoder(r.Body)
		var newJob allJob
		err := decoder.Decode(&newJob)
		if err != nil || len(newJob.Visits) != newJob.Count   {
			w.WriteHeader(400)
			m:=make(map[string]string)
			if err != nil {
				m["error"] = "Missing parameters"
			} else {
				m["error"] = "Count of visits is incorrect"
			}
			json.NewEncoder(w).Encode(m)
			return
		}


		for i:=0; i < len(newJob.Visits); i++ {
			if len(newJob.Visits[i].StoreId) == 0 || len(newJob.Visits[i].ImageUrl) ==0 || len(newJob.Visits[i].ImageUrl)==0{
				w.WriteHeader(400)
				m:=make(map[string]string)
				m["error"] = "Missing parameters"
				json.NewEncoder(w).Encode(m)
				return
			}
		}
		// get a unique ID of 20 chars
		jobid := genXid()
		_, err = repository.AddStatus(jobid, job)
		if err!=nil {
			w.WriteHeader(500)
			return
		}

		go processImage(job, newJob.Visits, jobid)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		m:=make(map[string]string)
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
			m:=make(map[string]string)
			m["error"] = "Incorrect Parameter"
			json.NewEncoder(w).Encode(m)
			return
		}

		key := jobid[0]
		key = string(key)
		status, err := repository.CheckStatus(key, job)


		if err!=nil{
			w.WriteHeader(400)
			m:=make(map[string]string)
			m["error"] = "jobid does not exist"
			json.NewEncoder(w).Encode(m)
			return
		}

		if status==0 || status == 1{
			w.WriteHeader(200)
			m:=make(map[string]string)
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
			for _, id := range storelist{
				storeID := StoreIdError{Error: ""}
				storeID.StoreID = id
				m.Error = append(m.Error, storeID)
			}
			json.NewEncoder(w).Encode(m)
			return
		}

	}
}

