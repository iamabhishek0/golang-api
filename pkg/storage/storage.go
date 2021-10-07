package storage

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

const defaultSchema = `CREATE TABLE IF NOT EXISTS users(id SERIAL PRIMARY KEY,name TEXT,email TEXT,phone TEXT)`

type storage struct {
	db *sql.DB
}

type Storage interface {
	Close() error
	DB() *sql.DB
}

func NewStorage() (Storage, error) {
	var err error

	s := new(storage)

	s.db, err = sql.Open("mysql", "root:n4ez4y2Fou2tMcqe@tcp(127.0.0.1:3306)/store")
	if err != nil {
		log.Println("Error while connecting to repository ", err)
		return nil, err
	}
	log.Println("Connected to repository")
	s.db.Exec(defaultSchema)
	return &storage{s.db}, nil
}

func (s *storage) DB() *sql.DB {
	return s.db
}

func (s *storage) Close() error {
	return s.db.Close()
}
