package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"backend/db"
	"backend/models"
	"backend/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetProducts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection("products").Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var products []models.Product
	cursor.All(ctx, &products)

	utils.JSON(w, http.StatusOK, products)
}

func GetProductByBarcode(w http.ResponseWriter, r *http.Request) {
	barcode := r.URL.Query().Get("barcode")

	var product models.Product
	err := db.DB.Collection("products").
		FindOne(context.Background(), bson.M{"barcode": barcode}).
		Decode(&product)

	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	utils.JSON(w, http.StatusOK, product)
}

// API-compatible handlers expected by the frontend
func GetProductsAPI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection("products").Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var products []models.Product
	cursor.All(ctx, &products)

	// Wrap response to match frontend `{ success, products }`
	resp := map[string]interface{}{
		"success":  true,
		"products": products,
	}
	utils.JSON(w, http.StatusOK, resp)
}

// ProductAPIHandler handles dynamic product subpaths:
// /api/product/<barcode>
// /api/product/<barcode>/macros
// /api/product/<barcode>/recommendations
// /api/product/<barcode>/recipes
func ProductAPIHandler(w http.ResponseWriter, r *http.Request) {
	prefix := "/api/product/"
	path := ""
	if len(r.URL.Path) > len(prefix) {
		path = r.URL.Path[len(prefix):]
	}
	// path may be "<barcode>" or "<barcode>/macros" etc
	parts := strings.SplitN(path, "/", 2)
	barcode := parts[0]
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}

	// find product in DB (if exists)
	var productMap bson.M
	_ = db.DB.Collection("products").FindOne(context.Background(), bson.M{"barcode": barcode}).Decode(&productMap)

	switch sub {
	case "macros":
		// try to extract macros from stored raw_data or fields
		// default to zeros
		macros := map[string]interface{}{"calories_kcal": 0, "protein_g": 0, "carbs_g": 0, "fat_g": 0, "per": "100g"}
		if raw, ok := productMap["raw_data"].(string); ok && raw != "" {
			var rawObj map[string]interface{}
			if err := json.Unmarshal([]byte(raw), &rawObj); err == nil {
				if nutr, ok := rawObj["nutriments"].(map[string]interface{}); ok {
					if v, ok := nutr["energy-kcal_100g"].(float64); ok {
						macros["calories_kcal"] = v
					}
					if v, ok := nutr["proteins_100g"].(float64); ok {
						macros["protein_g"] = v
					}
					if v, ok := nutr["carbohydrates_100g"].(float64); ok {
						macros["carbs_g"] = v
					}
					if v, ok := nutr["fat_100g"].(float64); ok {
						macros["fat_g"] = v
					}
				}
			}
		}
		utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "macros": macros})
		return
	case "recommendations":
		// simple DB-based recommendations: products with higher ecoScore
		var recommendations []bson.M
		score := 0
		if v, ok := productMap["ecoScore"].(float64); ok {
			score = int(v)
		}
		if v, ok := productMap["EcoScore"].(float64); ok {
			score = int(v)
		}
		cursor, err := db.DB.Collection("products").Find(context.Background(), bson.M{"ecoScore": bson.M{"$gt": score}}, options.Find().SetLimit(6))
		if err == nil {
			cursor.All(context.Background(), &recommendations)
		}
		// map to frontend expectation
		dbProds := make([]map[string]interface{}, 0)
		for _, r := range recommendations {
			dbProds = append(dbProds, map[string]interface{}{
				"name":        r["name"],
				"brand":       r["brand"],
				"barcode":     r["barcode"],
				"green_score": r["ecoScore"],
			})
		}
		resp := map[string]interface{}{"success": true, "recommendations": map[string]interface{}{"database_products": dbProds, "ai_suggestions": []interface{}{}, "current_score": score, "improvement_tips": []string{}}}
		utils.JSON(w, http.StatusOK, resp)
		return
	case "recipes":
		// Simple recipe generator using product name and raw_data ingredients
		recipes := []map[string]interface{}{}
		prodName := "Product"
		if n, ok := productMap["name"].(string); ok && n != "" {
			prodName = n
		} else if n, ok := productMap["product_name"].(string); ok && n != "" {
			prodName = n
		}

		// try to extract ingredients list from raw_data
		ingredients := []string{}
		if raw, ok := productMap["raw_data"].(string); ok && raw != "" {
			var rawObj map[string]interface{}
			if err := json.Unmarshal([]byte(raw), &rawObj); err == nil {
				if ingr, ok := rawObj["ingredients_text"].(string); ok && ingr != "" {
					// split on commas for a simple list
					parts := strings.Split(ingr, ",")
					for _, p := range parts {
						s := strings.TrimSpace(p)
						if s != "" {
							ingredients = append(ingredients, s)
						}
					}
				}
			}
		}

		// Build two simple recipes
		recipes = append(recipes, map[string]interface{}{
			"title":        prodName + " Quick Stir-Fry",
			"time_minutes": 20,
			"ingredients": func() []string {
				base := []string{prodName, "olive oil", "salt", "pepper", "garlic"}
				if len(ingredients) > 0 {
					base = append(base, ingredients[:min(len(ingredients), 3)]...)
				}
				return base
			}(),
			"steps": []string{
				"Heat oil in a pan.",
				"Add " + prodName + " and stir-fry for 5-8 minutes.",
				"Season with salt and pepper, add garlic and cook 1 more minute.",
				"Serve hot over rice or noodles.",
			},
		})

		recipes = append(recipes, map[string]interface{}{
			"title":        "Roasted " + prodName + " with Herbs",
			"time_minutes": 35,
			"ingredients": func() []string {
				base := []string{prodName, "olive oil", "rosemary", "thyme", "lemon"}
				if len(ingredients) > 0 {
					base = append(base, ingredients[:min(len(ingredients), 2)]...)
				}
				return base
			}(),
			"steps": []string{
				"Preheat oven to 200°C (400°F).",
				"Toss " + prodName + " with oil, herbs, salt, and lemon.",
				"Roast for 20-30 minutes until golden.",
				"Serve warm with a side salad.",
			},
		})

		utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "recipes": recipes})
		return
	default:
		// return product
		if productMap == nil {
			utils.JSON(w, http.StatusOK, map[string]interface{}{"success": false})
			return
		}
		utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "product": productMap})
		return
	}
}

// Add or update a product in the products collection
func AddProductAPI(w http.ResponseWriter, r *http.Request) {
	var p models.Product
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	// Try to update by barcode, otherwise insert
	filter := bson.M{"barcode": p.Barcode}
	update := bson.M{"$set": p}
	opts := options.Update().SetUpsert(true)

	_, err = db.DB.Collection("products").UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		http.Error(w, "Failed to save product", http.StatusInternalServerError)
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true})
}
