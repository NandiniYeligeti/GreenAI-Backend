package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"backend/db"
	"backend/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GoalsHandler routes GET and POST for /api/goals
func GoalsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getGoals(w, r)
	case http.MethodPost:
		createGoal(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func getGoals(w http.ResponseWriter, r *http.Request) {
	cursor, err := db.DB.Collection("goals").Find(context.Background(), bson.M{})
	if err != nil {
		utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "goals": []interface{}{}})
		return
	}
	var results []bson.M
	cursor.All(context.Background(), &results)
	for i := range results {
		if oid, ok := results[i]["_id"].(primitive.ObjectID); ok {
			results[i]["id"] = oid.Hex()
		}
	}
	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "goals": results})
}

func createGoal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type        string  `json:"type"`
		Description string  `json:"description"`
		TargetValue float64 `json:"target_value"`
		Progress    float64 `json:"progress"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	record := bson.M{
		"type":         req.Type,
		"description":  req.Description,
		"target_value": req.TargetValue,
		"progress":     req.Progress,
		"created_at":   time.Now(),
	}
	res, err := db.DB.Collection("goals").InsertOne(context.Background(), record)
	if err != nil {
		http.Error(w, "Failed to create goal", http.StatusInternalServerError)
		return
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		record["id"] = oid.Hex()
	}
	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "goal": record})
}
