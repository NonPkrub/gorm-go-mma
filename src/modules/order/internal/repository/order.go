package repository

import (
	"context"
	"database/sql"
	"fmt"
	"go-mma/modules/order/internal/model"
	"go-mma/shared/common/errs"
	"go-mma/shared/common/storage/gorm"
	"time"
)

// --> Step 1: สร้าง interface
type OrderRepository interface {
	Create(ctx context.Context, order *model.Order) error
	FindByID(ctx context.Context, id int64) (*model.Order, error)
	Cancel(ctx context.Context, id int64) error
}

type orderRepository struct { // --> Step 2: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
	dbCtx gorm.DBContext
}

// --> Step 3: return เป็น interface
func NewOrderRepository(dbCtx gorm.DBContext) OrderRepository {
	return &orderRepository{ // --> Step 4: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
		dbCtx: dbCtx,
	}
}

func (r *orderRepository) Create(ctx context.Context, m *model.Order) error {
	//Gorm
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx := r.dbCtx.DB().WithContext(ctx).Model(&model.Order{}).Create(&m)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *orderRepository) FindByID(ctx context.Context, id int64) (*model.Order, error) {
	//Gorm
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var order model.Order

	if err := r.dbCtx.DB().WithContext(ctx).Where("id = ?", id).First(&order).Error; err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errs.HandleDBError(fmt.Errorf("an error occurred while finding a order by id: %w", err))
	}
	return &order, nil
}

func (r *orderRepository) Cancel(ctx context.Context, id int64) error {
	//Gorm
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := r.dbCtx.DB().WithContext(ctx).Where("id = ?", id).Delete(&model.Order{}).Error; err != nil {
		return errs.HandleDBError(fmt.Errorf("failed to cancel order: %w", err))
	}
	return nil
}
