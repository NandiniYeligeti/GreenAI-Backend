package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"backend/db"
	"backend/utils"

	"go.mongodb.org/mongo-driver/bson"
)

// GetImpactStats reads aggregated impact totals and recent weekly numbers
func GetImpactStats(w http.ResponseWriter, r *http.Request) {
	var impactDoc bson.M
	_ = db.DB.Collection("impact").FindOne(context.Background(), bson.M{"_id": "global"}).Decode(&impactDoc)

	totalCarbon := 0.0
	if v, ok := impactDoc["total_carbon_saved"].(float64); ok {
		totalCarbon = v
	}

	// include total_baskets and total_score if present
	totalBaskets := 0
	if v, ok := impactDoc["total_baskets"].(int32); ok {
		totalBaskets = int(v)
	} else if v, ok := impactDoc["total_baskets"].(int64); ok {
		totalBaskets = int(v)
	}

	totalScore := 0.0
	if v, ok := impactDoc["total_score"].(float64); ok {
		totalScore = v
	} else if v, ok := impactDoc["total_score"].(int32); ok {
		totalScore = float64(v)
	} else if v, ok := impactDoc["total_score"].(int64); ok {
		totalScore = float64(v)
	}

	// compute weekly report: sum baskets in last 7 days
	weekAgo := time.Now().AddDate(0, 0, -7)
	cursor, err := db.DB.Collection("baskets").Find(context.Background(), bson.M{"created_at": bson.M{"$gte": weekAgo}})
	weeklySum := 0.0
	if err == nil {
		var docs []bson.M
		cursor.All(context.Background(), &docs)
		for _, d := range docs {
			if v, ok := d["total_carbon"].(float64); ok {
				weeklySum += v
			}
		}
	}

	avgScore := 0.0
	if totalBaskets > 0 {
		avgScore = totalScore / float64(totalBaskets)
	}

	stats := map[string]interface{}{
		"total_carbon_saved": totalCarbon,
		"total_baskets":      totalBaskets,
		"total_score":        totalScore,
		"average_score":      fmtFloat(avgScore),
		"weekly_report":      "You reduced your carbon footprint by " + fmtFloat(weeklySum) + " kg this week",
		"active_goals":       []interface{}{},
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "stats": stats})
}

// GetBadges returns user badges from DB
func GetBadges(w http.ResponseWriter, r *http.Request) {
	cursor, err := db.DB.Collection("user_badges").Find(context.Background(), bson.M{})
	if err != nil {
		utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "badges": []interface{}{}})
		return
	}
	var badges []bson.M
	cursor.All(context.Background(), &badges)

	utils.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "badges": badges})
}

func fmtFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 1, 64)
}
