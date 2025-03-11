package service

import (
	"errors"
	"fmt"
	"github.com/xuri/excelize/v2"
	"log"
	"railway/dao"
	"strconv"
	"strings"
)

var RailWayService dao.RailWayDAO

type OriginalRailWay struct {
	TrainNumber     string
	TrainNo         string
	StationSequence string
	ArrivalStation  string
	ArrivalTime     string
	DepartureTime   string
	RunningTime     string
	ArrivalDay      string
	StopTime        int64
}

func DownLoadRailWay() error {
	file, err := excelize.OpenFile("train_schedule.xlsx")
	if err != nil {
		log.Fatalf("无法打开文件: %v", err)
		return err
	}
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

	original := make([]OriginalRailWay, 0)
	railWays := make([]*dao.RailWay, 0)
	sum := 0
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
		originalRailway := OriginalRailWay{
			TrainNumber:     row[0],
			TrainNo:         row[1],
			StationSequence: row[2],
			ArrivalStation:  row[3],
			ArrivalTime:     row[4],
			DepartureTime:   row[5],
			RunningTime:     row[6],
			ArrivalDay:      row[7],
		}
		if len(original) == 0 || originalRailway.TrainNumber != original[0].TrainNumber {
			originalRailway.StopTime = 0
			original = make([]OriginalRailWay, 0) //清空
			original = append(original, originalRailway)
		} else {
			originalRailway.StopTime = CalculateStopTime(originalRailway.ArrivalTime, originalRailway.DepartureTime)
			for _, frontOriginalRailway := range original {
				railWay := dao.RailWay{
					TrainNumber:      frontOriginalRailway.TrainNumber,
					TrainNo:          frontOriginalRailway.TrainNo,
					DepartureStation: frontOriginalRailway.ArrivalStation,
					DepartureTime:    frontOriginalRailway.DepartureTime,
					ArrivalStation:   originalRailway.ArrivalStation,
					ArrivalTime:      originalRailway.ArrivalTime,
				}
				startTime, err := GetTime(frontOriginalRailway.RunningTime)
				if err != nil {
					return err
				}
				arriveTime, err := GetTime(originalRailway.RunningTime)
				if err != nil {
					return err
				}
				runningTime := arriveTime - startTime - frontOriginalRailway.StopTime
				//fmt.Println(railWay)
				//fmt.Printf("arriveTime :%d startTime :%d runningTime: %d\n", arriveTime, startTime, runningTime)
				//fmt.Printf("%s %s\n", originalRailway.ArrivalTime, originalRailway.DepartureTime)
				//time.Sleep(1)
				railWay.RunningTime = TurnToTime(runningTime)
				if runningTime < 1440 {
					railWay.ArrivalDay = 0
				} else if runningTime < 2880 {
					railWay.ArrivalDay = 1
				} else {
					railWay.ArrivalDay = 2
				}
				railWays = append(railWays, &railWay)
				sum = sum + 1
				if sum%1000 == 0 {
					fmt.Println(sum)
					err = RailWayService.BatchCreateRailWays(railWays)
					if err != nil {
						return err
					}
					railWays = make([]*dao.RailWay, 0)
				}
			}
			original = append(original, originalRailway)
		}
	}
	err = RailWayService.BatchCreateRailWays(railWays)
	if err != nil {
		return err
	}
	fmt.Println("railway create success")
	return nil
}

func GetTime(inputTime string) (int64, error) {
	timeString := strings.Split(inputTime, ":")
	if len(timeString) != 2 {
		return 0, errors.New("[GetTime] 时间输入不符合规范")
	}
	hours, err := strconv.ParseInt(timeString[0], 10, 64)
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.ParseInt(timeString[1], 10, 64)
	if err != nil {
		return 0, err
	}
	return hours*60 + minutes, nil
}
func OtherDay(inputDay string) int64 {
	switch inputDay {
	case "当日到达":
		return 0
	case "次日到达":
		return 1440
	case "第三日到达":
		return 2880
	default:
		return -2880
	}
}

func TurnToTime(inputTime int64) string {
	hours := inputTime / 60
	minutes := inputTime % 60
	return strconv.Itoa(int(hours)) + ":" + strconv.Itoa(int(minutes))
}

func CalculateStopTime(aTime, dTime string) int64 {
	oDTime, _ := GetTime(dTime)
	oATime, _ := GetTime(aTime)
	if oDTime == oATime {
		return 0
	}
	return (oDTime - oATime + 1440) % 1440
}
