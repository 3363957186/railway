package service

import (
	"container/heap"
	"errors"
	"fmt"
	"github.com/xuri/excelize/v2"
	"log"
	"math"
	"railway/dao"
	"sort"
	"strconv"
	"strings"
)

const (
	Default       = 0
	OnlyHighSpeed = 1
	OnlyLowSpeed  = 2

	LowRunningTimeFirst  = 1
	HighRunningTimeFirst = 2

	EarlyFirst = 3
	LateFirst  = 4

	LowPriceFirst  = 5
	HighPriceFirst = 6

	GetAllResult    = 1
	DefaultStopTime = 15

	StartIndex = "Start"
	EndIndex   = "End"
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

type TemplateTrainSchedule struct {
	ID               []uint
	TrainNumber      []string
	TrainNo          []string
	DepartureStation []string
	DepartureTime    []string
	ArrivalStation   []string
	ArrivalTime      []string
	RunningTime      []string
	ArrivalDay       []uint
	IsHighSpeed      []uint
	AllRunningTime   uint
}

type AnalyseTrans struct {
	NowTrainNumber  string
	NowTrainNo      string
	NowStation      string
	NowStatus       string //所在点是Arrive点还是Depart点
	TrainNumber     []string
	TrainNo         []string
	StationSequence []string
	AllRunningTime  int64
	TransFerTimes   int64 //中转次数
	NowArrivalDay   int64 //目前所在第几天
}

type RailwayService interface {
	SearchDirectly(departureStation, arrivalStation string, speedOption, sortOption int) ([]dao.RailWay, error)
	SearchDirectlyOnline(departureStation, arrivalStation string) ([]dao.RailWay, error)
	SearchWithOneTrans(departureStation, arrivalStation string, speedOption, sortOption int, limitStopTime, getAllResult int64) (map[string][]dao.RailWay, error)
	SearchWithTwoTrans(departureStation, arrivalStation string, speedOption, sortOption int, limitStopTime, getAllResult int64) (map[string][]dao.RailWay, error)
}

type RailWayServiceImpl struct {
	RailWayDAO dao.RailWayDAO
	StationDAO dao.StationDAO
}

var (
	RailWayDAO dao.RailWayDAO
	_          RailwayService = (*RailWayServiceImpl)(nil)
	Graph                     = make(map[string][]dao.RailWay) //图，前面的string是图中的点，以D或者A开头（表示出发还是到达）加上站点名加上车次NO加上第几天的车；后面的[]是从这个点出发的边
	//在站内转乘时TrainNumber记为Waiting，TrainNo为arrival的TrainNo，这样能够找到下一班车所在点
)

func NewRailwayService(RailWayDAO dao.RailWayDAO, StationDAO dao.StationDAO) RailWayServiceImpl {
	return RailWayServiceImpl{
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
	result = resultDedUp(result)
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

func (r *RailWayServiceImpl) SearchWithOneTrans(departureStation, arrivalStation string, speedOption, sortOption int, limitStopTime, getAllResult int64) (map[string][]dao.RailWay, error) {
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
	result := CombineTrainSchedule(departTrain, arrivalTrain, speedOption)
	return SortTransResult(result, sortOption, limitStopTime, getAllResult), nil
}

func (r *RailWayServiceImpl) SearchWithTwoTrans(departureStation, arrivalStation string, speedOption, sortOption int, limitStopTime, getAllResult int64) (map[string][]dao.RailWay, error) {
	return nil, nil
}

// 对转乘的进行排序，并且根据排序结果分别取动车前10，普车前10，动普混合前10
func SortTransResult(result map[string][]dao.RailWay, sortOption int, limitStopTime int64, getAllResult int64) map[string][]dao.RailWay {
	highSpeed := make(map[string][]dao.RailWay)
	lowSpeed := make(map[string][]dao.RailWay)
	highAndLow := make(map[string][]dao.RailWay)
	allOptions := make(map[string][]dao.RailWay)
	templateStruct := make([]TemplateTrainSchedule, 0)
	for _, tr := range result {
		templateStruct = append(templateStruct, ChangeToTemplate(tr, limitStopTime))
	}
	switch sortOption {
	case LowRunningTimeFirst:
		templateStruct = sortTemplateStructByLowRunningTime(templateStruct)
	case HighRunningTimeFirst:
		templateStruct = sortTemplateStructByHighRunningTime(templateStruct)
	case EarlyFirst:
		templateStruct = sortTemplateStructByEarlyFirst(templateStruct)
	case LateFirst:
		templateStruct = sortTemplateStructByLateFirst(templateStruct)
	default:
		templateStruct = sortTemplateStructByLowRunningTime(templateStruct)
	}
	if getAllResult == GetAllResult {
		for _, tr := range templateStruct {
			railways, trainString := ChangeToRailWays(tr)
			highSpeed[trainString] = railways
		}
		return highSpeed
	}
	for _, tr := range templateStruct {
		railways, trainString := ChangeToRailWays(tr)
		highSpeedCount := 0
		for _, railway := range railways {
			highSpeedCount = highSpeedCount + int(railway.IsHighSpeed)
		}
		if len(railways) == highSpeedCount && len(highSpeed) < 10 {
			highSpeed[trainString] = railways
			allOptions[trainString] = railways
		} else if highSpeedCount == 0 && len(highAndLow) < 10 {
			highAndLow[trainString] = railways
			allOptions[trainString] = railways
		} else if len(lowSpeed) < 10 {
			lowSpeed[trainString] = railways
			allOptions[trainString] = railways
		}
	}
	return allOptions
}

func ChangeToRailWays(tt TemplateTrainSchedule) ([]dao.RailWay, string) {
	result := make([]dao.RailWay, 0)
	trainString := ""
	for index, _ := range tt.ID {
		trainString = trainString + tt.TrainNumber[index] + "/"
		result = append(result, dao.RailWay{
			ID:               tt.ID[index],
			TrainNumber:      tt.TrainNumber[index],
			TrainNo:          tt.TrainNo[index],
			ArrivalStation:   tt.ArrivalStation[index],
			ArrivalTime:      tt.ArrivalTime[index],
			DepartureStation: tt.DepartureStation[index],
			DepartureTime:    tt.DepartureTime[index],
			RunningTime:      tt.RunningTime[index],
			ArrivalDay:       tt.ArrivalDay[index],
			IsHighSpeed:      tt.IsHighSpeed[index],
		})
	}
	trainString = trainString + strconv.Itoa(int(tt.AllRunningTime))
	return result, trainString
}

func ChangeToTemplate(railWays []dao.RailWay, limitStopTime int64) TemplateTrainSchedule {
	schedule := TemplateTrainSchedule{
		ID:               make([]uint, 0),
		TrainNumber:      make([]string, 0),
		TrainNo:          make([]string, 0),
		DepartureStation: make([]string, 0),
		DepartureTime:    make([]string, 0),
		ArrivalStation:   make([]string, 0),
		ArrivalTime:      make([]string, 0),
		RunningTime:      make([]string, 0),
		ArrivalDay:       make([]uint, 0),
		IsHighSpeed:      make([]uint, 0),
		AllRunningTime:   0,
	}
	for _, train := range railWays {
		schedule.ID = append(schedule.ID, train.ID)
		schedule.TrainNumber = append(schedule.TrainNumber, train.TrainNumber)
		schedule.TrainNo = append(schedule.TrainNo, train.TrainNo)
		schedule.DepartureStation = append(schedule.DepartureStation, train.DepartureStation)
		schedule.DepartureTime = append(schedule.DepartureTime, train.DepartureTime)
		schedule.ArrivalStation = append(schedule.ArrivalStation, train.ArrivalStation)
		schedule.ArrivalTime = append(schedule.ArrivalTime, train.ArrivalTime)
		schedule.RunningTime = append(schedule.RunningTime, train.RunningTime)
		schedule.ArrivalDay = append(schedule.ArrivalDay, train.ArrivalDay)
		schedule.IsHighSpeed = append(schedule.IsHighSpeed, train.IsHighSpeed)
	}
	schedule.AllRunningTime = uint(GetAllRunningTime(schedule, limitStopTime))
	return schedule
}
func GetTransTime(arrivalTime, departureTime string, limitStopTime int64) int64 {
	oDTime, _ := GetTime(departureTime)
	oATime, _ := GetTime(arrivalTime)
	StopTime := oDTime - oATime
	if StopTime < limitStopTime {
		StopTime = StopTime + 1440 //换乘时间小于最短时间，默认加1天
	}
	return StopTime
}

func GetAllRunningTime(templateTrainSchedule TemplateTrainSchedule, limitStopTime int64) int64 {
	allTime := int64(0)
	for index, runningTime := range templateTrainSchedule.RunningTime {
		intRunningTime, _ := GetTime(runningTime)
		transTime := int64(0)
		if index != 0 {
			transTime = GetTransTime(templateTrainSchedule.ArrivalTime[index-1], templateTrainSchedule.DepartureTime[index], limitStopTime)
		}
		allTime = allTime + intRunningTime + transTime
	}
	return allTime
}

func sortTemplateStructByLowRunningTime(result []TemplateTrainSchedule) []TemplateTrainSchedule {
	sort.Slice(result, func(i, j int) bool {
		if result[i].AllRunningTime == result[j].AllRunningTime {
			iDepartTime, _ := GetTime(result[i].DepartureTime[0])
			jDepartTime, _ := GetTime(result[j].DepartureTime[0])
			return iDepartTime < jDepartTime
		}
		return result[i].AllRunningTime < result[j].AllRunningTime
	})
	return result
}

func sortTemplateStructByHighRunningTime(result []TemplateTrainSchedule) []TemplateTrainSchedule {
	sort.Slice(result, func(i, j int) bool {
		if result[i].AllRunningTime == result[j].AllRunningTime {
			iDepartTime, _ := GetTime(result[i].DepartureTime[0])
			jDepartTime, _ := GetTime(result[j].DepartureTime[0])
			return iDepartTime < jDepartTime
		}
		return result[i].AllRunningTime > result[j].AllRunningTime
	})
	return result
}

func sortTemplateStructByEarlyFirst(result []TemplateTrainSchedule) []TemplateTrainSchedule {
	sort.Slice(result, func(i, j int) bool {
		iDepartTime, _ := GetTime(result[i].DepartureTime[0])
		jDepartTime, _ := GetTime(result[j].DepartureTime[0])
		if iDepartTime == jDepartTime {
			return result[i].AllRunningTime < result[j].AllRunningTime
		}
		return iDepartTime < jDepartTime
	})
	return result
}

func sortTemplateStructByLateFirst(result []TemplateTrainSchedule) []TemplateTrainSchedule {
	sort.Slice(result, func(i, j int) bool {
		iDepartTime, _ := GetTime(result[i].DepartureTime[0])
		jDepartTime, _ := GetTime(result[j].DepartureTime[0])
		if iDepartTime == jDepartTime {
			return result[i].AllRunningTime < result[j].AllRunningTime
		}
		return iDepartTime > jDepartTime
	})
	return result
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

func sortByEarlyArriveFirst(result []dao.RailWay) []dao.RailWay {
	sort.Slice(result, func(i, j int) bool {
		iArrivalTime, _ := GetTime(result[i].ArrivalTime)
		jArrivalTime, _ := GetTime(result[j].ArrivalTime)
		if iArrivalTime == jArrivalTime {
			iRunningTime, _ := GetTime(result[i].RunningTime)
			jRunningTime, _ := GetTime(result[j].RunningTime)
			return iRunningTime < jRunningTime
		}
		return iArrivalTime < jArrivalTime
	})
	return result
}

func CombineTrainSchedule(departTrain, arrivalTrain []dao.RailWay, speedOption int) map[string][]dao.RailWay {
	result := make(map[string][]dao.RailWay)
	for _, dT := range departTrain {
		if speedOption == OnlyHighSpeed && dT.IsHighSpeed == 0 {
			continue
		}
		if speedOption == OnlyLowSpeed && dT.IsHighSpeed == 1 {
			continue
		}
		for _, aT := range arrivalTrain {
			if speedOption == OnlyHighSpeed && aT.IsHighSpeed == 0 {
				continue
			}
			if speedOption == OnlyLowSpeed && aT.IsHighSpeed == 1 {
				continue
			}
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

func resultDedUp(result []dao.RailWay) []dao.RailWay {
	mapResult := make(map[string]dao.RailWay)
	dedUpResult := make([]dao.RailWay, 0)
	for _, train := range result {
		mapResult[train.TrainNumber] = train
	}
	for _, train := range mapResult {
		dedUpResult = append(dedUpResult, train)
	}
	return dedUpResult
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
	mapOriginal := make(map[string]OriginalRailWay)
	railWays := make([]dao.RailWay, 0)
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
		if len(original) == 0 || originalRailway.TrainNo != original[0].TrainNo {
			originalRailway.StopTime = 0
			original = make([]OriginalRailWay, 0) //清空
			mapOriginal = make(map[string]OriginalRailWay)
			original = append(original, originalRailway)
			mapOriginal[original[0].ArrivalStation] = originalRailway
		} else {
			_, ok := mapOriginal[originalRailway.ArrivalStation]
			if ok {
				continue
			}
			if originalRailway.StationSequence == "01" {
				originalRailway.StopTime = 0
			} else {
				originalRailway.StopTime = CalculateStopTime(originalRailway.ArrivalTime, originalRailway.DepartureTime)
			}
			for _, frontOriginalRailway := range original {
				tFrontOriginalRailway := frontOriginalRailway
				tOriginalRailway := originalRailway
				if tFrontOriginalRailway.StationSequence > tOriginalRailway.StationSequence {
					tFrontOriginalRailway, tOriginalRailway = tOriginalRailway, tFrontOriginalRailway
				}
				railWay := dao.RailWay{
					TrainNumber:      tFrontOriginalRailway.TrainNumber,
					TrainNo:          tFrontOriginalRailway.TrainNo,
					DepartureStation: tFrontOriginalRailway.ArrivalStation,
					DepartureTime:    tFrontOriginalRailway.DepartureTime,
					ArrivalStation:   tOriginalRailway.ArrivalStation,
					ArrivalTime:      tOriginalRailway.ArrivalTime,
				}
				if railWay.TrainNumber[0] == 'G' || railWay.TrainNumber[0] == 'D' || railWay.TrainNumber[0] == 'C' {
					railWay.IsHighSpeed = 1
				} else {
					railWay.IsHighSpeed = 0
				}
				startTime, err := GetTime(tFrontOriginalRailway.RunningTime)
				if err != nil {
					return err
				}
				arriveTime, err := GetTime(tOriginalRailway.RunningTime)
				if err != nil {
					return err
				}
				runningTime := arriveTime - startTime - tFrontOriginalRailway.StopTime
				if runningTime < 0 {
					fmt.Printf("runningTime<0 tFrontOriginalRailway:%v tOriginalRailway:%v\n", tFrontOriginalRailway, tOriginalRailway)
					continue
				}

				railWay.RunningTime = TurnToTime(runningTime)
				if runningTime < 1440 {
					railWay.ArrivalDay = 0
				} else if runningTime < 2880 {
					railWay.ArrivalDay = 1
				} else {
					railWay.ArrivalDay = 2
				}
				railWays = append(railWays, railWay)
				sum = sum + 1
				if sum%100 == 0 {
					if sum%10000 == 0 {
						fmt.Println(sum)
					}
					err = RailWayDAO.BatchCreateRailWays(railWays)
					if err != nil {
						return err
					}
					railWays = make([]dao.RailWay, 0)
				}
			}
			original = append(original, originalRailway)
			mapOriginal[originalRailway.ArrivalStation] = originalRailway
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
	return strconv.FormatInt(hours, 10) + ":" + strconv.FormatInt(minutes, 10)
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

func (r *RailWayServiceImpl) AddNewStation(stationName string, isDeparture bool, startTime int64) error {
	if isDeparture {
		departureTrains, err := r.RailWayDAO.GetRailWayByDepartureStation(stationName)
		if err != nil {
			log.Fatal(err)
			return err
		}
		departureTrains = getKeyTrains(departureTrains, true, 0)
		for _, train := range departureTrains {
			dTime, _ := GetTime(train.DepartureTime)
			if startTime >= dTime {
				value, ok := Graph[StartIndex]
				if ok {
					value = append(value, train)
					Graph[StartIndex] = value
				} else {
					Graph[StartIndex] = []dao.RailWay{train}
				}
			}
		}
	} else {
		arrivalTrains, err := r.RailWayDAO.GetRailWayByArrivalStation(stationName)
		if err != nil {
			log.Fatal(err)
			return err
		}
		arrivalTrains = getKeyTrains(arrivalTrains, true, 0)
		getKeyTrains(arrivalTrains, true, 1)
		getKeyTrains(arrivalTrains, true, 2)
	}
	return nil
}

func (r *RailWayServiceImpl) InitBuildGraph() error {
	/**
	可以将构造图的模式调整成完全图，不使用KeyStation来构造图，这样构造出来的整个图是所有列车运行时刻表形成的图
	总计150w边+50w点，对于大型服务器来说，以这个数据量去跑最短路还是可行的，但是单台电脑不太行
	**/

	//allStation, err := r.StationDAO.GetAllStations()
	//if err != nil {
	//	return err
	//}
	//for _, station := range allStation {
	for key, _ := range KeyStation {
		//key := station.StationName
		//fmt.Println(key)
		arrivalTrains, err := r.RailWayDAO.GetRailWayByArrivalStation(key)
		if err != nil {
			log.Fatal(err)
			return err
		}
		departureTrains, err := r.RailWayDAO.GetRailWayByDepartureStation(key)
		if err != nil {
			log.Fatal(err)
			return err
		}
		departureTrains = sortByEarlyArriveFirst(departureTrains)
		arrivalTrains = sortByEarlyArriveFirst(arrivalTrains)

		arrivalTrains = getKeyTrains(arrivalTrains, true, 0)
		departureTrains = getKeyTrains(departureTrains, true, 0)
		getKeyTrains(arrivalTrains, true, 1)
		getKeyTrains(arrivalTrains, true, 2)
		getKeyTrains(departureTrains, true, 1)
		getKeyTrains(departureTrains, true, 2)

		departureTrains = sortByEarlyFirst(departureTrains)

		length := len(departureTrains)
		for index, train := range departureTrains {
			buildDepartureWaitingEdges(departureTrains[(index+1)%length], train, 0)
		}
		for index, train := range departureTrains {
			buildDepartureWaitingEdges(departureTrains[(index+1)%length], train, 1)
		}
		for index, train := range departureTrains {
			buildDepartureWaitingEdges(departureTrains[(index+1)%length], train, 2)
		}
		buildArrivalToDepartureWaitingEdges(arrivalTrains, departureTrains, DefaultStopTime)
	}
	log.Printf("[InitBuildGraph] Building graph successfully")
	return nil
}

func getKeyTrains(input []dao.RailWay, isAddGraph bool, dayTime int) []dao.RailWay {
	result := make([]dao.RailWay, 0)
	rememberTrainNo := make(map[string]string)
	for _, v := range input {
		_, ok := rememberTrainNo[v.TrainNo]
		if ok {
			continue
		}
		if checkKeyStation(v.DepartureStation) && checkKeyStation(v.ArrivalStation) {
			rememberTrainNo[v.TrainNo] = v.TrainNo
			result = append(result, v)
			if isAddGraph {
				//加边
				vv := v
				vv.ArrivalDay = vv.ArrivalDay + uint(dayTime)
				departIndex := "D/" + v.DepartureStation + "/" + v.TrainNo + "/" + strconv.Itoa(dayTime)
				value, ok := Graph[departIndex]
				if ok {
					value = append(value, vv)
					Graph[departIndex] = value
				} else {
					Graph[departIndex] = []dao.RailWay{vv}
				}
			}
		}
	}
	return result
}

// 只要满足一个是关键站点即可
func getOneKeyTrains(input []dao.RailWay, isAddGraph bool, dayTime int) []dao.RailWay {
	result := make([]dao.RailWay, 0)
	for _, v := range input {
		if checkKeyStation(v.DepartureStation) || checkKeyStation(v.ArrivalStation) {
			result = append(result, v)
			if isAddGraph {
				//加边
				vv := v
				vv.ArrivalDay = vv.ArrivalDay + uint(dayTime)
				departIndex := "D/" + v.DepartureStation + "/" + v.TrainNo + "/" + strconv.Itoa(dayTime)
				value, ok := Graph[departIndex]
				if ok {
					value = append(value, vv)
					Graph[departIndex] = value
				} else {
					Graph[departIndex] = []dao.RailWay{vv}
				}
			}
		}
	}
	return result
}

func buildDepartureWaitingEdges(arrival, departure dao.RailWay, arrivalDay int64) {
	newEdge := dao.RailWay{
		TrainNumber:      "Waiting",
		TrainNo:          arrival.TrainNo,
		DepartureStation: departure.DepartureStation,
		ArrivalStation:   arrival.DepartureTime,
		DepartureTime:    departure.DepartureTime,
		ArrivalTime:      arrival.DepartureTime,
		ArrivalDay:       uint(arrivalDay),
	}
	runningTime := CalculateStopTime(newEdge.ArrivalTime, newEdge.DepartureTime)
	newEdge.RunningTime = TurnToTime(runningTime)
	dTime, _ := GetTime(newEdge.DepartureTime)
	aTime, _ := GetTime(newEdge.ArrivalTime)
	if dTime > aTime {
		newEdge.ArrivalDay = newEdge.ArrivalDay + 1
	}
	departIndex := "D/" + newEdge.DepartureStation + "/" + departure.TrainNo + "/" + strconv.FormatInt(arrivalDay, 10)
	value, ok := Graph[departIndex]
	if ok {
		value = append(value, newEdge)
		Graph[departIndex] = value
	} else {
		Graph[departIndex] = []dao.RailWay{newEdge}
	}
	return
}

func checkKeyStation(stationName string) bool {
	_, ok := KeyStation[stationName]
	return ok
}
func buildArrivalToDepartureWaitingEdges(arrivalTrains, departureTrains []dao.RailWay, limitStopTime int64) {
	dIndex := 0
	dLength := len(departureTrains)
	if dLength == 0 {
		return
	}
	for _, arrival := range arrivalTrains {
		for _, departure := range departureTrains {
			if arrival.TrainNo == departure.TrainNo {
				turnADToEdges(arrival, departureTrains[dIndex], 0)
				turnADToEdges(arrival, departureTrains[dIndex], 1)
				turnADToEdges(arrival, departureTrains[dIndex], 2)
			}
		}
	}
	for _, arrival := range arrivalTrains {
		isSuccess := false
		for i := dIndex; i < dLength; i++ {
			aTime, _ := GetTime(arrival.ArrivalTime)
			dTime, _ := GetTime(departureTrains[dIndex].DepartureTime)
			if aTime+limitStopTime <= dTime {
				turnADToEdges(arrival, departureTrains[dIndex], 0)
				turnADToEdges(arrival, departureTrains[dIndex], 1)
				turnADToEdges(arrival, departureTrains[dIndex], 2)
				isSuccess = true
				dIndex = i
			}
			dIndex = i
		}
		if !isSuccess {
			turnADToEdges(arrival, departureTrains[0], 0)
			turnADToEdges(arrival, departureTrains[0], 1)
			turnADToEdges(arrival, departureTrains[0], 2)
		}
	}
}

func turnADToEdges(arrival, departure dao.RailWay, arrivalDay int64) {
	newEdge := dao.RailWay{
		TrainNumber:      "Waiting",
		TrainNo:          departure.TrainNo,
		DepartureStation: arrival.ArrivalStation,
		ArrivalStation:   departure.DepartureStation,
		DepartureTime:    arrival.ArrivalTime,
		ArrivalTime:      departure.DepartureTime,
		ArrivalDay:       uint(arrivalDay),
	}
	runningTime := CalculateStopTime(newEdge.ArrivalTime, newEdge.DepartureTime)
	newEdge.RunningTime = TurnToTime(runningTime)
	dTime, _ := GetTime(newEdge.DepartureTime)
	aTime, _ := GetTime(newEdge.ArrivalTime)
	if dTime > aTime {
		newEdge.ArrivalDay = newEdge.ArrivalDay + 1
	}
	arrivalIndex := "A/" + newEdge.DepartureStation + "/" + arrival.TrainNo + "/" + strconv.FormatInt(arrivalDay, 10)
	value, ok := Graph[arrivalIndex]
	if ok {
		value = append(value, newEdge)
		Graph[arrivalIndex] = value
	} else {
		Graph[arrivalIndex] = []dao.RailWay{newEdge}
	}
}

func Dijkstra(graph map[string][]Edge, start string) map[string]AnalyseTrans {
	// 初始化最短路径映射
	dist := make(map[string]AnalyseTrans)

	// 初始化所有点的路径值为最大
	for node := range graph {
		dist[node] = AnalyseTrans{
			AllRunningTime: math.MaxInt64,
			TransFerTimes:  math.MaxInt64,
		}
	}
	dist[start] = AnalyseTrans{
		AllRunningTime: 0,
		TransFerTimes:  0,
	}

	// 初始化最小堆
	pq := &PriorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &Item{node: start, allTime: 0, transferTimes: 0})

	// 运行 Dijkstra
	for pq.Len() > 0 {
		curr := heap.Pop(pq).(*Item)
		currNode, currTime, currTransfers := curr.node, curr.allTime, curr.transferTimes

		// 如果当前路径已经不是最短路径，则跳过
		if currTime > dist[currNode].AllRunningTime ||
			(currTime == dist[currNode].AllRunningTime && currTransfers > dist[currNode].TransFerTimes) {
			continue
		}

		// 遍历邻接点
		for _, edge := range graph[currNode] {
			nextNode, travelTime, transfers := edge.To, edge.AllRunningTime, edge.TransFerTimes
			newTime := currTime + travelTime
			newTransfers := currTransfers + transfers

			// 如果找到更优路径，则更新
			if newTime < dist[nextNode].AllRunningTime ||
				(newTime == dist[nextNode].AllRunningTime && newTransfers < dist[nextNode].TransFerTimes) {
				dist[nextNode] = AnalyseTrans{
					NowTrainNumber:  dist[currNode].NowTrainNumber,
					NowTrainNo:      dist[currNode].NowTrainNo,
					NowStation:      nextNode,
					NowStatus:       "Arrive",
					TrainNumber:     append(dist[currNode].TrainNumber, dist[currNode].NowTrainNumber),
					TrainNo:         append(dist[currNode].TrainNo, dist[currNode].NowTrainNo),
					StationSequence: append(dist[currNode].StationSequence, nextNode),
					AllRunningTime:  newTime,
					TransFerTimes:   newTransfers,
					NowArrivalDay:   dist[currNode].NowArrivalDay,
				}
				heap.Push(pq, &Item{node: nextNode, allTime: newTime, transferTimes: newTransfers})
			}
		}
	}

	return dist
}
