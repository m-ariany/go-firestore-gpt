package product

import (
	"go-firestore-gpt/internal/model"
)

type ProductEvent struct {
	Product model.Product
	Err     error
}
