package main

import (
	"fmt"
	"log"
	"os"
	"wbstorage/internal/db"
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {
	dbHost := getEnv("DB_HOST", "localhost")
	dbUser := getEnv("DB_USER", "user")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "dbname")

	dbConnString := fmt.Sprintf("user=%s dbname=%s sslmode=disable password=%s host=%s", dbUser, dbName, dbPassword, dbHost)
	dbConn, err := db.NewDB(dbConnString)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbConn.Db.Close()

	err = dbConn.CreateTables()

	if err != nil {
		log.Fatalf("failed to init database: %v", err)
	}

	log.Println("Successfully created tables")
}
