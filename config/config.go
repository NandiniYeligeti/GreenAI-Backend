package config

import "os"

var (
	MongoURI     = "mongodb://127.0.0.1:27017"
	DatabaseName = "greenlabelai"
	ServerPort   = ":8080"
)

func LoadEnv() {
	if uri := os.Getenv("MONGO_URI"); uri != "" {
		MongoURI = uri
	}
	if db := os.Getenv("DB_NAME"); db != "" {
		DatabaseName = db
	}
}
