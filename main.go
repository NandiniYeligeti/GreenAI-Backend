package main

import (
	"log"
	"net/http"

	"backend/config"
	"backend/db"
	"backend/routes"
)

func main() {
	config.LoadEnv()
	db.ConnectMongo()

	router := routes.RegisterRoutes()

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
