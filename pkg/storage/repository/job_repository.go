package repository

import (
	"golang-api/pkg/storage"
	"log"
)

const addJobStatus = "INSERT INTO job_status(jobid, status) VALUES(?, ?)"
const addImageData = "INSERT INTO image(jobid, storeid, perimeter) VALUES(?, ?, ?)"
const addFailedJob = "INSERT INTO failed(jobid, storeid) VALUES(?, ?)"
const updateJobStatus = "UPDATE job_status SET status=? WHERE jobid=?"
const checkStatus = "SELECT status FROM job_status WHERE jobid=?"
const getFailedStoreID = "SELECT storeid FROM failed WHERE jobid=?"

type JobStorage struct {
	Db storage.Storage
}

func NewJobRepository(dbStorage *storage.Storage) *JobStorage {
	return &JobStorage{Db: *dbStorage}
}

func AddStatus(jobid string, job JobStorage) (string, error) {
	_, err := job.Db.DB().Exec(addJobStatus, jobid, 0)
	if err!=nil{
		return "",nil
	}
	return jobid,nil
}

func AddImage(jobid string, storeid string, perimeter int, job JobStorage) (string, error) {
	_, err := job.Db.DB().Exec(addImageData, jobid, storeid, perimeter)
	if err!=nil{
		return "",nil
	}
	return jobid,nil
}

func CheckStatus(jobid string,job JobStorage) (int, error) {
	var status int;
	err := job.Db.DB().QueryRow(checkStatus, jobid).Scan(&status)
	if err!=nil{
		return 0,err
	}

	return status,nil
}

func AddFailed(jobid string, storeid string, job JobStorage)  (string, error) {
	_, err := job.Db.DB().Exec(addFailedJob, jobid, storeid)
	if err!=nil{
		return "",nil
	}
	return jobid,nil
}

func UpdateJobStatus(jobid string, status int, job JobStorage) (string, error) {
	_, err := job.Db.DB().Exec(updateJobStatus, status, jobid)
	if err!=nil{
		return "",nil
	}
	return jobid,nil
}

func GetFailedStoreId(jobid string, job JobStorage) ([]string, error){
	var storeid string
	var storelist []string
	rows, err := job.Db.DB().Query(getFailedStoreID, jobid)
	for rows.Next() {
		err := rows.Scan(&storeid)
		if err != nil {
			log.Fatal(err)
		}
		storelist = append(storelist, storeid)
	}
	return storelist, err
}