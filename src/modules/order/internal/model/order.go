package model

import (
	"go-mma/shared/common/idgen"
	"time"

	"gorm.io/gorm"
)

type Order struct {
	ID         int64          `db:"id"`
	CustomerID int64          `db:"customer_id"`
	OrderTotal int            `db:"order_total"`
	CreatedAt  time.Time      `db:"created_at"`
	CanceledAt gorm.DeletedAt `db:"canceled_at" gorm:"index"` // nullable
}

func (Order) TableName() string {
	return "sales.orders"
}

func NewOrder(customerID int64, orderTotal int) *Order {
	return &Order{
		ID:         idgen.GenerateTimeRandomID(),
		CustomerID: customerID,
		OrderTotal: orderTotal,
	}
}
