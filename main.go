package main

import (
	"fmt"
	"railway/mssql"
	"railway/service"
)

// 车站模型

func main() {
	//mssql.DropDB()
	mssql.InitStation()
	mssql.InitRailWay()
	//err := service.DownLoadStation()
	//if err != nil {
	//	fmt.Println(err)
	//}
	//err = service.DownLoadRailWay()
	//if err != nil {
	//	fmt.Println(err)
	//}
	RailwayService := service.NewRailwayService(service.RailWayDAO, service.StationService)
	result, err := RailwayService.SearchDirectly("杭州南", "上海松江", 0, 0)
	if err != nil {
		fmt.Println(err)
	}
	for _, record := range result {
		fmt.Println(record)
	}
	resultMap, err := RailwayService.SearchWithOneTrans("太原南", "上海", service.Default, service.Default, service.DefaultStopTime, service.DefaultStopTime)
	for key, value := range resultMap {
		fmt.Println(key, value)
	}
	result, err = RailwayService.SearchDirectly("淮安东", "太原南", 0, 0)
	if err != nil {
		fmt.Println(err)
	}
	for _, record := range result {
		fmt.Println(record)
	}
	result, err = RailwayService.SearchDirectly("郑州", "上海", 0, 0)
	if err != nil {
		fmt.Println(err)
	}
	for _, record := range result {
		fmt.Println(record)
	}
	//resultMap, err = RailwayService.SearchWithOneTrans("太原南", "上海", service.Default, service.Default, service.DefaultStopTime, service.GetAllResult)
	//for key, value := range resultMap {
	//	fmt.Println(key, value)
	//}
}
