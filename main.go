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
	err := service.DownLoadKeyStation()
	if err != nil {
		fmt.Println(err)
	}
	//err = service.DownLoadStation()
	//if err != nil {
	//	fmt.Println(err)
	//}
	//err = service.DownLoadRailWay()
	//if err != nil {
	//	fmt.Println(err)
	//}

	RailwayService := service.NewRailwayService(service.RailWayDAO, service.StationService)
	err = RailwayService.InitBuildGraph()
	if err != nil {
		fmt.Println(err)
	} else {
		sum := 0
		st := 0
		for _, value := range service.Graph {
			sum = sum + len(value)
			st = st + 1
		}
		fmt.Println(sum)
		fmt.Println(st)
	}
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
	result, err = RailwayService.SearchDirectly("杭州", "沈阳北", 0, 0)
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
	_, _ = RailwayService.SearchWithTwoTrans("杭州", "长白山", service.Default, service.Default, service.DefaultStopTime, service.Default)
	_, _ = RailwayService.SearchWithTwoTrans("乌鲁木齐", "海口", service.Default, service.Default, service.DefaultStopTime, service.Default)
}
