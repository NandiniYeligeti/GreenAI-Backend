package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Product struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Barcode     string             `bson:"barcode" json:"barcode"`
	EcoScore    int                `bson:"ecoScore" json:"ecoScore"`
	Description string             `bson:"description" json:"description"`
	ImageURL    string             `bson:"image_url,omitempty" json:"image_url"`
	Brand       string             `bson:"brand,omitempty" json:"brand"`
	RawData     string             `bson:"raw_data,omitempty" json:"raw_data"`
	CreatedAt   time.Time          `bson:"created_at,omitempty" json:"created_at"`
}
