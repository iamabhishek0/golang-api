package repository

import (
	"golang-api/pkg/storage"
)

const addJobStatus = "INSERT INTO job_status(jobid, status) VALUES(?, ?)"


type JobStorage struct {
	Db storage.Storage
}

func NewJobRepository(dbStorage *storage.Storage) *JobStorage {
	return &JobStorage{Db: *dbStorage}
}

func AddStatus(jobid string, job JobStorage) (string, error) {
	_, err := job.Db.DB().Exec(addJobStatus, jobid, false)
	if err!=nil{
		return "",nil
	}
	return jobid,nil
}