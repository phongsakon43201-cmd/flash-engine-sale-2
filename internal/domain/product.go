package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Product struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	ImageURL    string             `bson:"image_url" json:"image_url"`
	Price       float64            `bson:"price" json:"price"`
	Stock       int                `bson:"stock" json:"stock"` // Total DB stock
	IsFlashSale bool               `bson:"is_flash_sale" json:"is_flash_sale"`
	FlashPrice  float64            `bson:"flash_price,omitempty" json:"flash_price,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

type CreateProductDTO struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	ImageURL    string  `json:"image_url"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	IsFlashSale bool    `json:"is_flash_sale"`
	FlashPrice  float64 `json:"flash_price"`
}

type PrewarmStockDTO struct {
	ProductID string `json:"product_id"`
	Stock     int    `json:"stock"`
}
