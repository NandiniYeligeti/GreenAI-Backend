package routes

import (
	"net/http"

	"backend/handlers"
)

func RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/products", handlers.GetProducts)
	mux.HandleFunc("/product/barcode", handlers.GetProductByBarcode)

	// API routes expected by the frontend
	mux.HandleFunc("/api/products", handlers.GetProductsAPI)
	mux.HandleFunc("/api/product/", handlers.ProductAPIHandler) // will parse path after this prefix and handle subpaths
	mux.HandleFunc("/api/products/add", handlers.AddProductAPI)
	mux.HandleFunc("/api/basket", handlers.AnalyzeBasketAPI)
	mux.HandleFunc("/api/basket/save", handlers.SaveBasketAPI)
	mux.HandleFunc("/api/baskets", handlers.GetBasketsAPI)

	mux.HandleFunc("/basket", handlers.GetBasket)
	mux.HandleFunc("/basket/add", handlers.AddToBasket)

	mux.HandleFunc("/history", handlers.GetHistory)
	mux.HandleFunc("/history/clear", handlers.ClearHistory)
	mux.HandleFunc("/history/add", handlers.AddHistory)

	// Impact API endpoints
	mux.HandleFunc("/api/impact/stats", handlers.GetImpactStats)
	mux.HandleFunc("/api/badges", handlers.GetBadges)
	mux.HandleFunc("/api/goals", handlers.GoalsHandler)

	// CORS wrapper to allow frontend dev server access
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		mux.ServeHTTP(w, r)
	})

	return handler
}
