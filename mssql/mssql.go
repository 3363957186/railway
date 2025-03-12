package mssql

import (
	"fmt"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"log"
	"railway/dao"
	"railway/service"
)

var dsn = "sqlserver://sa:Zhangyi2002@localhost:1433?database=master"

func InitStation() {
	db, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}

	// 创建数据库（如果不存在的话）
	// 如果你想在 GORM 中直接创建数据库，你可以先连接到 `master` 数据库，然后创建目标数据库。
	// 你可以使用原生 SQL 来创建数据库。
	db.Exec("IF NOT EXISTS (SELECT * FROM sys.databases WHERE name = 'station_db') CREATE DATABASE station_db")

	// 切换到创建的数据库
	dsn = "sqlserver://sa:Zhangyi2002@localhost:1433?database=station_db" // 使用新的数据库
	db, err = gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}

	// 自动迁移：创建表格
	err = db.AutoMigrate(&dao.Station{})
	if err != nil {
		log.Fatalf("表格创建失败: %v", err)
	}
	fmt.Println("数据库和表格已成功创建或已存在！")
	service.StationService = dao.NewStationDAO(db)
}

func InitRailWay() {
	db, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}

	// 创建数据库（如果不存在的话）
	// 如果你想在 GORM 中直接创建数据库，你可以先连接到 `master` 数据库，然后创建目标数据库。
	// 你可以使用原生 SQL 来创建数据库。
	db.Exec("IF NOT EXISTS (SELECT * FROM sys.databases WHERE name = 'station_db') CREATE DATABASE station_db")

	// 切换到创建的数据库
	dsn = "sqlserver://sa:Zhangyi2002@localhost:1433?database=station_db" // 使用新的数据库
	db, err = gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}

	// 自动迁移：创建表格
	err = db.AutoMigrate(&dao.RailWay{})
	if err != nil {
		log.Fatalf("表格创建失败: %v", err)
	}
	fmt.Println("数据库和表格已成功创建或已存在！")
	service.RailWayDAO = dao.NewRailWayDAO(db)
}

func CleanRailWay() {
	db, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}
	dsn = "sqlserver://sa:Zhangyi2002@localhost:1433?database=station_db"
	err = db.Migrator().DropTable(&dao.RailWay{})
	if err != nil {
		log.Fatal(err)
	}
}

func CleanStation() {
	db, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}
	dsn = "sqlserver://sa:Zhangyi2002@localhost:1433?database=station_db"
	err = db.Migrator().DropTable(&dao.Station{})
	if err != nil {
		log.Fatal(err)
	}
}
