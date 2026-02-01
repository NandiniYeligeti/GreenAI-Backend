package main

import (
	"log"
	"net/http"
	"os"

	"backend/config"
	"backend/db"
	"backend/routes"
)

func main() {
	config.LoadEnv()
	db.ConnectMongo()

	router := routes.RegisterRoutes()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // local development fallback
	}

	log.Println("Server running on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
