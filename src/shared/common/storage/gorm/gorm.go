package gorm

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type closeDB func() error

type DBContext interface {
	DB() *gorm.DB
}

type dbContext struct {
	db *gorm.DB
}

var _ DBContext = (*dbContext)(nil)

func NewDBContext(dsn string) (DBContext, closeDB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	return &dbContext{db: db},
		func() error {
			return sqlDB.Close()
		},
		nil

}
func (c *dbContext) DB() *gorm.DB {
	return c.db
}
