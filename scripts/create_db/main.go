package main

import (
	"log"
	"wbstorage/internal/db"
)

func main() {
	dbConn, err := db.NewDB("user=felix dbname=wbstore sslmode=disable password=12345678 host=localhost")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbConn.Db.Close()

	dbConn.CreateTables()
}
