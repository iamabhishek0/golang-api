package rest

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"golang-api/pkg/storage/repository"
	"image"
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

func processImage(data []Job) bool{
	log.Println(len(data))
	for i:=0; i < len(data); i++ {

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
				log.Fatal(err)
			}
			g := m.Bounds()

			// Get height and width
			height := g.Dy()
			width := g.Dx()

			// The resolution is height x width
			resolution := 2 * (height + width)

			// Print results
			log.Println(height, width, resolution, "pixels")
		}
	}
	return true
}

func addJob(job repository.JobStorage) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		decoder := json.NewDecoder(r.Body)
		var newJob allJob
		err := decoder.Decode(&newJob)

		log.Println(newJob.Visits)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, _ = repository.AddStatus("122121", job)
		go processImage(newJob.Visits)
		w.Header().Set("Content-Type", "application/json")
		list := [7]string{"This", "is", "the", "tutorial","of", "Go", "language"}
		json.NewEncoder(w).Encode(list)
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

		log.Println("Url Param 'key' is: " + string(key))
		list := [7]string{"This", "is", "the", "tutorial","of", "Go", "language"}
		json.NewEncoder(w).Encode(list)
	}
}

