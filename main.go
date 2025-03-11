package main

import (
	"fmt"
	"railway/mssql"
	"railway/service"
)

// 车站模型

func main() {
	mssql.InitStation()
	mssql.InitRailWay()
	err := service.DownLoadStation()
	if err != nil {
		fmt.Println(err)
	}
	err = service.DownLoadRailWay()
	if err != nil {
		fmt.Println(err)
	}
}
