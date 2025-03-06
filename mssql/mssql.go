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

type Station struct {
	ID                 uint   `gorm:"primaryKey"`
	StationAbbr        string `gorm:"size:20"`  // 车站简称
	StationName        string `gorm:"size:100"` // 车站名
	StationCode        string `gorm:"size:20"`  // 车站代号
	StationPinyin      string `gorm:"size:100"` // 车站拼音
	StationFirstLetter string `gorm:"size:10"`  // 车站首字母
	StationNumber      string `gorm:"size:20"`  // 车站标号
	CityCode           string `gorm:"size:10"`  // 城市代码
	CityName           string `gorm:"size:100"` // 车站所属城市
}

func Init() {
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
	err = db.AutoMigrate(&Station{})
	if err != nil {
		log.Fatalf("表格创建失败: %v", err)
	}
	fmt.Println("数据库和表格已成功创建或已存在！")
	service.StationService = dao.NewStationDAO(db)
}
