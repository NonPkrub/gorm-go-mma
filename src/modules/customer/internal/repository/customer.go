package repository

import (
	"context"
	"database/sql"
	"fmt"
	"go-mma/modules/customer/internal/model"
	"go-mma/shared/common/errs"
	"go-mma/shared/common/storage/gorm"
	"time"
)

// --> Step 1: สร้าง interface
type CustomerRepository interface {
	Create(ctx context.Context, customer *model.Customer) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	FindByID(ctx context.Context, id int64) (*model.Customer, error)
	UpdateCredit(ctx context.Context, customer *model.Customer) error
}

type customerRepository struct { // --> Step 2: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
	dbCtx gorm.DBContext
}

// --> Step 3: return เป็น interface
func NewCustomerRepository(dbCtx gorm.DBContext) CustomerRepository {
	return &customerRepository{ // --> Step 4: เปลี่ยนชื่อ struct เป็นตัวพิมพ์เล็ก
		dbCtx: dbCtx,
	}
}

func (r *customerRepository) Create(ctx context.Context, customer *model.Customer) error {
	//Gorm
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	fmt.Println(customer)

	if err := r.dbCtx.DB().WithContext(ctx).Model(&model.Customer{}).Create(&customer).Error; err != nil {
		return errs.HandleDBError(fmt.Errorf("an error occurred while inserting customer: %w", err))
	}
	return nil
}

func (r *customerRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	//Gorm
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var exists int64
	if err := r.dbCtx.DB().WithContext(ctx).Model(&model.Customer{}).Where("email = ?", email).Count(&exists).Error; err != nil {
		return false, errs.HandleDBError(fmt.Errorf("an error occurred while checking email: %w", err))
	}
	return exists > 0, nil // ถ้าไม่ error แสดงว่ามี email ในระบบแล้ว
}

func (r *customerRepository) FindByID(ctx context.Context, id int64) (*model.Customer, error) {

	//Gorm
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var customer model.Customer
	if err := r.dbCtx.DB().WithContext(ctx).Model(&model.Customer{}).Where("id = ?", id).First(&customer).Error; err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errs.HandleDBError(fmt.Errorf("an error occurred while finding a customer by id: %w", err))
	}
	return &customer, nil
}

func (r *customerRepository) UpdateCredit(ctx context.Context, m *model.Customer) error {
	//Gorm
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if err := r.dbCtx.DB().WithContext(ctx).Model(&model.Customer{}).Where("id = ?", m.ID).Update("credit", m.Credit).Error; err != nil {
		return errs.HandleDBError(fmt.Errorf("an error occurred while updating customer credit: %w", err))
	}
	return nil
}
