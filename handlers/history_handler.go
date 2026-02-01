package handlers

import (
	"context"
	"net/http"
	"time"

	"backend/db"
	"backend/models"
	"backend/utils"
)

func AddHistory(w http.ResponseWriter, r *http.Request) {
	barcode := r.URL.Query().Get("barcode")

	history := models.ScanHistory{
		Barcode: barcode,
		Time:    time.Now(),
	}

	db.DB.Collection("history").InsertOne(context.Background(), history)
	utils.JSON(w, http.StatusCreated, history)
}

func GetHistory(w http.ResponseWriter, r *http.Request) {
	cursor, err := db.DB.Collection("history").Find(context.Background(), map[string]interface{}{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var history []models.ScanHistory
	cursor.All(context.Background(), &history)

	// Return wrapped response for frontend compatibility
	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "history": history})
}

func ClearHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, err := db.DB.Collection("history").DeleteMany(context.Background(), map[string]interface{}{})
	if err != nil {
		http.Error(w, "Failed to clear history", http.StatusInternalServerError)
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true})
}
