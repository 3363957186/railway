package service

import (
	"errors"
	"fmt"
	"github.com/xuri/excelize/v2"
	"log"
	"os"
	"railway/dao"
	"strings"
)

var StationService dao.StationDAO

var KeyStation map[string]dao.Station

func DownLoadStation() error {
	file, err := excelize.OpenFile("车站信息.xlsx")
	if err != nil {
		log.Fatalf("无法打开文件: %v", err)
		return err
	}

	// 获取第一个工作表名称
	sheetNames := file.GetSheetList()
	if len(sheetNames) == 0 {
		log.Fatal("Excel 文件中没有工作表")
		return errors.New("Excel 文件中没有工作表")
	}

	sheetName := file.GetSheetName(0)

	// 读取 Excel 工作表的数据
	rows, err := file.GetRows(sheetName)
	if err != nil {
		log.Fatalf("无法读取工作表数据: %v", err)
		return err
	}

	// 创建一个切片，用于保存转换后的 Station 结构体
	var stations []dao.Station

	// 跳过表头，读取每一行数据并将其转换为 Station 结构体
	for i, row := range rows {
		if i == 0 {
			// 跳过表头
			continue
		}

		// 确保每行有 8 列数据
		if len(row) < 8 {
			continue
		}

		// 将每行数据转换为 Station 结构体
		station := dao.Station{
			StationAbbr:        row[0], // 车站简称
			StationName:        row[1], // 车站名
			StationCode:        row[2], // 车站代号
			StationPinyin:      row[3], // 车站拼音
			StationFirstLetter: row[4], // 车站首字母
			StationNumber:      row[5], // 车站标号
			CityCode:           row[6], // 城市代码
			CityName:           row[7], // 车站所属城市
			IsKeyStation:       0,      //非二次转乘站点
		}

		// 将转换后的 Station 加入切片
		stations = append(stations, station)
	}

	// 打印所有读取到的车站数据
	for _, station := range stations {
		err := StationService.CreateStation(&station)
		if err != nil {
			fmt.Printf("station:%v, err:%s\n", station, err)
			return err
		}
		//else {
		//	fmt.Printf("index %d success!\n", index)
		//}
	}
	data, err := os.ReadFile("站点选择.txt") // 确保文件路径正确
	if err != nil {
		fmt.Println("读取文件失败:", err)
		return err
	}

	// 按换行符分割字符串
	cities := strings.Split(string(data), "\n")

	// 去掉可能的空行
	for _, city := range cities {
		city = strings.TrimSpace(city)
		Stations, err := StationService.GetStationByCityName(city)
		if err != nil {
			fmt.Printf("city:%v, err:%s\n", city, err)
			return err
		}
		for _, station := range Stations {
			station.IsKeyStation = 1
			KeyStation[station.StationName] = station
			err = StationService.UpdateStation(&station)
		}
	}

	log.Print("DownLoadStation success\n")
	return nil
}
