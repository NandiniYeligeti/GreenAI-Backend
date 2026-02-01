package models

import "time"

type ScanHistory struct {
	Barcode string    `bson:"barcode" json:"barcode"`
	Time    time.Time `bson:"time" json:"time"`
}
