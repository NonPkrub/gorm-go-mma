package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go-mma/application"
	"go-mma/config"
	"go-mma/modules/customer"
	"go-mma/modules/notification"
	"go-mma/modules/order"
	"go-mma/shared/common/logger"
	"go-mma/shared/common/module"
	"go-mma/shared/common/storage/gorm"
	"go-mma/shared/common/storage/gorm/transactor"
)

var (
	Version = "local-dev"
	Time    = "n/a"
)

func main() {
	closeLog, err := logger.Init()
	if err != nil {
		panic(err.Error())
	}
	defer closeLog()

	config, err := config.Load()
	if err != nil {
		panic(err.Error())
	}

	dbCtx, closeDB, err := gorm.NewDBContext(config.DSN)
	if err != nil {
		panic(err.Error())
	}
	defer func() { // ใช่ท่า IIFE เพราะต้องการแสดง error ถ้าปิดไม่ได้
		if err := closeDB(); err != nil {
			logger.Log.Error(fmt.Sprintf("Error closing database: %v", err))
		}
	}()

	app := application.New(*config)

	transactor, _ := transactor.New(dbCtx.DB(),
		transactor.WithNestedTransactionStrategy(transactor.NestedTransactionsSavepoints),
	)

	mCtx := module.NewModuleContext(transactor, dbCtx)

	app.RegisterModules(
		notification.NewModule(mCtx),
		customer.NewModule(mCtx),
		order.NewModule(mCtx),
	)

	app.Run()

	// รอสัญญาณการปิด
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Log.Info("Shutting down...")

	app.Shutdown()

	// Optionally: แล้วค่อยปิด resource อื่นๆ เช่น DB connection, cleanup, etc.

	logger.Log.Info("Shutdown complete.")
}
