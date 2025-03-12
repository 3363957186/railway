package service

import (
	"errors"
	"fmt"
	"github.com/xuri/excelize/v2"
	"log"
	"railway/dao"
	"sort"
	"strconv"
	"strings"
)

const (
	OnlyHighSpeed = 1
	OnlyLowSpeed  = 2

	LowRunningTimeFirst  = 1
	HighRunningTimeFirst = 2

	EarlyFirst = 3
	LateFirst  = 4

	LowPriceFirst  = 5
	HighPriceFirst = 6
)

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

type RailwayService interface {
	SearchDirectly(departureStation, arrivalStation string, speedOption, sortOption int) ([]dao.RailWay, error)
	SearchDirectlyOnline(departureStation, arrivalStation string) ([]dao.RailWay, error)
	SearchWithOneTrans(departureStation, arrivalStation string, speedOption, sortOption int) (map[string][]dao.RailWay, error)
}

type RailWayServiceImpl struct {
	RailWayDAO dao.RailWayDAO
	StationDAO dao.StationDAO
}

var (
	RailWayDAO dao.RailWayDAO
	_          RailwayService = (*RailWayServiceImpl)(nil)
)

func NewRailwayService(RailWayDAO dao.RailWayDAO, StationDAO dao.StationDAO) RailwayService {
	return &RailWayServiceImpl{
		RailWayDAO: RailWayDAO,
		StationDAO: StationDAO,
	}
}

func (r *RailWayServiceImpl) SearchDirectly(departureStation, arrivalStation string, speedOption, sortOption int) (result []dao.RailWay, err error) {
	if !r.checkStation(departureStation) || !r.checkStation(arrivalStation) {
		log.Printf("[SearchDirectly] stationNotFind")
		return nil, errors.New("stationNotFind")
	}
	switch speedOption {
	case OnlyHighSpeed:
		result, err = r.RailWayDAO.GetRailWayByDepartureStationAndArrivalStationOnlyHighSpeed(departureStation, arrivalStation)
	case OnlyLowSpeed:
		result, err = r.RailWayDAO.GetRailWayByDepartureStationAndArrivalStationOnlyLowSpeed(departureStation, arrivalStation)
	default:
		result, err = r.RailWayDAO.GetRailWayByDepartureStationAndArrivalStation(departureStation, arrivalStation)
	}
	if err != nil {
		log.Printf("[SearchDirectly] err:%s", err.Error())
		return nil, err
	}
	switch sortOption {
	case LowRunningTimeFirst:
		result = sortByLowRunningTime(result)
	case HighRunningTimeFirst:
		result = sortByHighRunningTime(result)
	case EarlyFirst:
		result = sortByEarlyFirst(result)
	case LateFirst:
		result = sortByLateFirst(result)
	default:
		result = sortByEarlyFirst(result)
	}
	return result, nil
}
func (r *RailWayServiceImpl) SearchDirectlyOnline(departureStation, arrivalStation string) ([]dao.RailWay, error) {
	return nil, errors.New("not implement")
}

func (r *RailWayServiceImpl) SearchWithOneTrans(departureStation, arrivalStation string, speedOption, sortOption int) (map[string][]dao.RailWay, error) {
	if !r.checkStation(departureStation) || !r.checkStation(arrivalStation) {
		log.Printf("[SearchDirectly] stationNotFind")
		return nil, errors.New("stationNotFind")
	}
	departTrain, err := r.RailWayDAO.GetRailWayByDepartureStationWithoutArrivalStation(departureStation, arrivalStation)
	if err != nil {
		log.Printf("[SearchWithOneTrans ] err:%s", err.Error())
		return nil, err
	}
	arrivalTrain, err := r.RailWayDAO.GetRailWayByArrivalStationWithoutDepartureStation(departureStation, arrivalStation)
	if err != nil {
		log.Printf("[SearchWithOneTrans ] err:%s", err.Error())
		return nil, err
	}
	return CombineTrainSchedule(departTrain, arrivalTrain), nil
}

func (r *RailWayServiceImpl) checkStation(stationName string) bool {
	station, err := r.StationDAO.GetStationByName(stationName)
	if err != nil {
		fmt.Println(err)
		log.Printf("[checkStation] error err:%s\n", err.Error())
		return false
	}
	if station == nil || station.StationName != stationName {
		return false
	}
	return true
}

func sortByLowRunningTime(result []dao.RailWay) []dao.RailWay {
	sort.Slice(result, func(i, j int) bool {
		iRunningTime, _ := GetTime(result[i].RunningTime)
		jRunningTime, _ := GetTime(result[j].RunningTime)
		if iRunningTime == jRunningTime {
			iDepartTime, _ := GetTime(result[i].DepartureTime)
			jDepartTime, _ := GetTime(result[j].DepartureTime)
			return iDepartTime < jDepartTime
		}
		return iRunningTime < jRunningTime
	})
	return result
}

func sortByHighRunningTime(result []dao.RailWay) []dao.RailWay {
	sort.Slice(result, func(i, j int) bool {
		iRunningTime, _ := GetTime(result[i].RunningTime)
		jRunningTime, _ := GetTime(result[j].RunningTime)
		if iRunningTime == jRunningTime {
			iDepartTime, _ := GetTime(result[i].DepartureTime)
			jDepartTime, _ := GetTime(result[j].DepartureTime)
			return iDepartTime < jDepartTime
		}
		return iRunningTime > jRunningTime
	})
	return result
}

func sortByEarlyFirst(result []dao.RailWay) []dao.RailWay {
	sort.Slice(result, func(i, j int) bool {
		iDepartTime, _ := GetTime(result[i].DepartureTime)
		jDepartTime, _ := GetTime(result[j].DepartureTime)
		if iDepartTime == jDepartTime {
			iRunningTime, _ := GetTime(result[i].RunningTime)
			jRunningTime, _ := GetTime(result[j].RunningTime)
			return iRunningTime < jRunningTime
		}
		return iDepartTime < jDepartTime
	})
	return result
}

func sortByLateFirst(result []dao.RailWay) []dao.RailWay {
	sort.Slice(result, func(i, j int) bool {
		iDepartTime, _ := GetTime(result[i].DepartureTime)
		jDepartTime, _ := GetTime(result[j].DepartureTime)
		if iDepartTime == jDepartTime {
			iRunningTime, _ := GetTime(result[i].RunningTime)
			jRunningTime, _ := GetTime(result[j].RunningTime)
			return iRunningTime < jRunningTime
		}
		return iDepartTime > jDepartTime
	})
	return result
}

func CombineTrainSchedule(departTrain, arrivalTrain []dao.RailWay) map[string][]dao.RailWay {
	result := make(map[string][]dao.RailWay)
	for _, dT := range departTrain {
		for _, aT := range arrivalTrain {
			if dT.ArrivalStation == aT.DepartureStation {
				title := dT.TrainNumber + aT.TrainNumber
				value, ok := result[title]
				if ok {
					if len(value) == 0 {
						result[title] = []dao.RailWay{dT, aT}
					} else {
						//只在最后一个可以换乘的站进行换乘
						if value[0].RunningTime < dT.RunningTime {
							result[title] = []dao.RailWay{dT, aT}
						}
					}
				} else {
					result[title] = []dao.RailWay{dT, aT}
				}
			}
		}
	}
	return result
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
				if railWay.TrainNumber[0] == 'G' || railWay.TrainNumber[0] == 'D' || railWay.TrainNumber[0] == 'C' {
					railWay.IsHighSpeed = 1
				} else {
					railWay.IsHighSpeed = 0
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
					err = RailWayDAO.BatchCreateRailWays(railWays)
					if err != nil {
						return err
					}
					railWays = make([]*dao.RailWay, 0)
				}
			}
			original = append(original, originalRailway)
		}
	}
	err = RailWayDAO.BatchCreateRailWays(railWays)
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

func CompareTime(aTime, dTime string) bool {
	oDTime, _ := GetTime(dTime)
	oATime, _ := GetTime(aTime)
	if oATime >= oDTime {
		return true
	}
	return false
}
