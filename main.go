package main

import (
	"fmt"
	"railway/mssql"
	"railway/service"
	"railway/web"
)

func init() {
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
	service.R = service.NewRailwayService(service.RailWayDAO, service.StationService)
	web.H = web.NewHandler(service.R)
}

// 车站模型

func main() {

	err := service.R.InitBuildGraph()
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
	result, err := service.R.SearchDirectly("杭州南", "上海松江", "all", 0)
	if err != nil {
		fmt.Println(err)
	}
	for _, record := range result {
		fmt.Println(record)
	}
	resultMap, err := service.R.SearchWithOneTrans("太原南", "上海", service.Default, 0, service.DefaultStopTime, service.DefaultStopTime)
	for key, value := range resultMap {
		fmt.Println(key, value)
	}
	result, err = service.R.SearchDirectly("淮安东", "太原南", service.Default, 0)
	if err != nil {
		fmt.Println(err)
	}
	for _, record := range result {
		fmt.Println(record)
	}
	result, err = service.R.SearchDirectly("杭州", "沈阳北", service.Default, 0)
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
	resultMap, err = service.R.SearchWithTwoTrans("杭州东", "长白山", service.Default, 3, 5)
	if err != nil {
		fmt.Println(err)
	}
	for key, value := range resultMap {
		fmt.Println(key, value)

	}
	resultMap, err = service.R.SearchWithTwoTrans("乌鲁木齐", "海口", service.Default, 3, 5)
	if err != nil {
		fmt.Println(err)
	}
	for key, value := range resultMap {
		fmt.Println(key, value)
	}
	web.StartNgork()
}
