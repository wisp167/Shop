package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
)

type Models struct {
	Shop ShopModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Shop: ShopModel{DB: db},
	}
}
