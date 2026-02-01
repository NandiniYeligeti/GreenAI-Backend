package models

type BasketItem struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}
