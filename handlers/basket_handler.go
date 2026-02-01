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
	"go.mongodb.org/mongo-driver/mongo/options"
)

var basket []interface{}

func AddToBasket(w http.ResponseWriter, r *http.Request) {
	var item map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	basket = append(basket, item)
	utils.JSON(w, http.StatusOK, basket)
}

func GetBasket(w http.ResponseWriter, r *http.Request) {
	utils.JSON(w, http.StatusOK, basket)
}

// AnalyzeBasketAPI accepts { barcodes: string[] } and returns simple aggregated stats
func AnalyzeBasketAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Barcodes []string `json:"barcodes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	type Item struct {
		Barcode     string  `json:"barcode"`
		ProductName string  `json:"product_name"`
		Carbon      float64 `json:"carbon"`
		HealthScore int     `json:"health_score"`
	}

	var items []Item
	var totalCarbon float64
	var totalHealth int

	for _, code := range req.Barcodes {
		var prod map[string]interface{}
		err := db.DB.Collection("products").FindOne(context.Background(), bson.M{"barcode": code}).Decode(&prod)
		eco := 50
		name := ""
		if err == nil {
			if v, ok := prod["ecoScore"].(float64); ok {
				eco = int(v)
			} else if v, ok := prod["ecoScore"].(int); ok {
				eco = v
			} else if v, ok := prod["EcoScore"].(float64); ok {
				eco = int(v)
			}
			if n, ok := prod["name"].(string); ok {
				name = n
			} else if n, ok := prod["Name"].(string); ok {
				name = n
			}
		}

		carbon := (100 - float64(eco)) * 0.05 // arbitrary formula
		items = append(items, Item{Barcode: code, ProductName: name, Carbon: carbon, HealthScore: eco})
		totalCarbon += carbon
		totalHealth += eco
	}

	avgHealth := 0
	if len(items) > 0 {
		avgHealth = totalHealth / len(items)
	}

	resp := map[string]interface{}{
		"success": true,
		"basket": map[string]interface{}{
			"total_items":      len(items),
			"total_carbon":     totalCarbon,
			"avg_health_score": avgHealth,
			"items":            items,
		},
	}
	utils.JSON(w, http.StatusOK, resp)
}

// SaveBasketAPI saves the analyzed basket into the `baskets` collection
func SaveBasketAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Barcodes []string `json:"barcodes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	type Item struct {
		Barcode     string  `json:"barcode"`
		ProductName string  `json:"product_name"`
		Carbon      float64 `json:"carbon"`
		HealthScore int     `json:"health_score"`
	}

	var items []Item
	var totalCarbon float64
	var totalHealth int

	for _, code := range req.Barcodes {
		var prod map[string]interface{}
		err := db.DB.Collection("products").FindOne(context.Background(), bson.M{"barcode": code}).Decode(&prod)
		eco := 50
		name := ""
		if err == nil {
			if v, ok := prod["ecoScore"].(float64); ok {
				eco = int(v)
			} else if v, ok := prod["ecoScore"].(int); ok {
				eco = v
			} else if v, ok := prod["EcoScore"].(float64); ok {
				eco = int(v)
			}
			if n, ok := prod["name"].(string); ok {
				name = n
			} else if n, ok := prod["Name"].(string); ok {
				name = n
			}
		}

		carbon := (100 - float64(eco)) * 0.05
		items = append(items, Item{Barcode: code, ProductName: name, Carbon: carbon, HealthScore: eco})
		totalCarbon += carbon
		totalHealth += eco
	}

	avgHealth := 0
	if len(items) > 0 {
		avgHealth = totalHealth / len(items)
	}

	record := bson.M{
		"barcodes":         req.Barcodes,
		"items":            items,
		"total_items":      len(items),
		"total_carbon":     totalCarbon,
		"avg_health_score": avgHealth,
		"created_at":       time.Now(),
	}

	res, err := db.DB.Collection("baskets").InsertOne(context.Background(), record)
	if err != nil {
		http.Error(w, "Failed to save basket", http.StatusInternalServerError)
		return
	}

	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		record["id"] = oid.Hex()
	}

	// Update impact totals (global single document)
	// include total_score so the app can track cumulative basket scores
	inc := bson.M{"$inc": bson.M{"total_carbon_saved": record["total_carbon"], "total_baskets": 1, "total_score": record["avg_health_score"]}}
	setOnInsert := bson.M{"$setOnInsert": bson.M{"created_at": time.Now()}}
	update := bson.M{"$setOnInsert": setOnInsert["$setOnInsert"], "$inc": inc["$inc"]}
	// Using options to upsert
	_, _ = db.DB.Collection("impact").UpdateOne(context.Background(), bson.M{"_id": "global"}, update, options.Update().SetUpsert(true))

	// Award badges based on thresholds
	// Simple badge rules:
	// 1 - First Basket (total_baskets >= 1)
	// 2 - Carbon Saver (total_carbon_saved >= 10)
	// 3 - Super Saver (total_carbon_saved >= 100)

	var impactDoc bson.M
	_ = db.DB.Collection("impact").FindOne(context.Background(), bson.M{"_id": "global"}).Decode(&impactDoc)
	totalCarbonSaved := 0.0
	if v, ok := impactDoc["total_carbon_saved"].(float64); ok {
		totalCarbonSaved = v
	} else if v, ok := impactDoc["total_carbon_saved"].(int32); ok {
		totalCarbonSaved = float64(v)
	}

	// read cumulative total_score (sum of avg_health_score across saved baskets)
	totalScore := 0.0
	if v, ok := impactDoc["total_score"].(float64); ok {
		totalScore = v
	} else if v, ok := impactDoc["total_score"].(int32); ok {
		totalScore = float64(v)
	}
	// read total_baskets for potential badge rules
	totalBaskets := int64(0)
	if v, ok := impactDoc["total_baskets"].(int32); ok {
		totalBaskets = int64(v)
	} else if v, ok := impactDoc["total_baskets"].(int64); ok {
		totalBaskets = v
	}

	// Helper to award badge if not already awarded
	awardBadge := func(badgeID int, name, desc string) {
		// check if exists
		count, _ := db.DB.Collection("user_badges").CountDocuments(context.Background(), bson.M{"badge_id": badgeID})
		if count == 0 {
			db.DB.Collection("user_badges").InsertOne(context.Background(), bson.M{"badge_id": badgeID, "badge": bson.M{"id": badgeID, "name": name, "description": desc}, "earned_at": time.Now()})
		}
	}

	// Always award First Basket if not present
	awardBadge(1, "First Basket", "Saved your first basket")
	if totalCarbonSaved >= 10 {
		awardBadge(2, "Carbon Saver", "Saved 10kg CO2 or more")
	}
	if totalCarbonSaved >= 100 {
		awardBadge(3, "Super Saver", "Saved 100kg CO2 or more")
	}

	// Score / consistency based badges
	// 4 - Consistent Shopper (saved >= 10 baskets)
	if totalBaskets >= 10 {
		awardBadge(4, "Consistent Shopper", "Saved 10 baskets")
	}
	// 5 - Healthy Shopper (cumulative score threshold)
	if totalScore >= 500 {
		awardBadge(5, "Healthy Shopper", "Accumulated 500+ health score")
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "basket": record})
}

// GetBasketsAPI returns saved baskets (most recent first)
func GetBasketsAPI(w http.ResponseWriter, r *http.Request) {
	cursor, err := db.DB.Collection("baskets").Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, "Failed to fetch baskets", http.StatusInternalServerError)
		return
	}
	var results []bson.M
	cursor.All(context.Background(), &results)

	for i := range results {
		if oid, ok := results[i]["_id"].(primitive.ObjectID); ok {
			results[i]["id"] = oid.Hex()
		}
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "baskets": results})
}
