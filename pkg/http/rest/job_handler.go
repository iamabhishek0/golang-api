package rest

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/xid"
	repository "golang-api/pkg/storage/repository"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
)

type Job struct {
	Store_id   string   `json:"store_id"`
	Image_url  []string `json:"image_url"`
	Visit_time string   `json:"visit_time"`
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
	Status string `json:"status"`
	Job_id string         `json:"job_id"`
	Error  []StoreIdError `json:"error"`
}

func genXid() string{
	id := xid.New()
	return id.String()
}

func processImage(job repository.JobStorage, data []Job, jobid string) bool{
	log.Println(len(data))
	var status = 1
	for i:=0; i < len(data); i++ {
		storeid := data[i].Store_id

		for j:=0; j < len(data[i].Image_url); j++ {
			path:=data[i].Image_url[j]

			log.Println(path)
			resp, err := http.Get(path)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()
			m, _, err := image.Decode(resp.Body)
			if err != nil {
				status = 2
				repository.AddFailed(jobid, storeid, job)
				break;
				//log.Fatal(err)
			}
			g := m.Bounds()

			// Get height and width
			height := g.Dy()
			width := g.Dx()

			// The resolution is height x width
			perimeter := 2 * (height + width)

			_, _ = repository.AddImage(jobid, storeid, perimeter, job)
			if err != nil {
				return false
			}
			// Print results
			//log.Println(height, width, resolution, "pixels")
		}
	}

	repository.UpdateJobStatus(jobid, status, job)
	return true
}

func addJob(job repository.JobStorage) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		decoder := json.NewDecoder(r.Body)
		var newJob allJob
		err := decoder.Decode(&newJob)
		log.Println(newJob)
		if err != nil || len(newJob.Visits) != newJob.Count   {
			w.WriteHeader(400)
			m:=make(map[string]string)
			m["error"] = ""
			json.NewEncoder(w).Encode(m)
			return
		}

		jobid := genXid()
		_, _ = repository.AddStatus(jobid, job)
		go processImage(job, newJob.Visits, jobid)


		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		m:=make(map[string]string)
		m["jobid"] = jobid
		log.Println("jobid ",m["jobid"])
		json.NewEncoder(w).Encode(m)
	}
}


func getJobStatus(job repository.JobStorage) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		jobid, ok := r.URL.Query()["jobid"]

		if !ok || len(jobid[0]) < 1 {
			log.Println("Url Param 'key' is missing")
			return
		}

		// Query()["key"] will return an array of items,
		// we only want the single item.
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

			m := FailedStoreID{Job_id: key, Status: "failed"}
			for _, id := range storelist{
				storeID := StoreIdError{Error: ""}
				storeID.StoreID = id
				m.Error = append(m.Error, storeID)
			}
			log.Println(m)
			json.NewEncoder(w).Encode(m)
			return
		}

	}
}

