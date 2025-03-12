package main

import (
	"fmt"
	"railway/mssql"
	"railway/service"
)

// 车站模型

func main() {
	//mssql.CleanRailWay()
	//mssql.CleanStation()
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
}
