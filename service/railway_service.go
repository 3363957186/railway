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
	Default       = "all"
	OnlyHighSpeed = "highspeed"
	OnlyLowSpeed  = "normal"

	DefaultSort          = 0
	LowRunningTimeFirst  = 3
	HighRunningTimeFirst = 4

	EarlyFirst = 5
	LateFirst  = 6

	LowPriceFirst  = 1
	HighPriceFirst = 2

	GetAllResult    = 1
	DefaultStopTime = 15

	DefaultResultNumber = 10

	StartIndex = "Start"
	EndIndex   = "End"
	Waiting    = "Waiting"
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
	ToTalPrice      float64
	TransFerTimes   int64 //中转次数
	NowArrivalDay   int64 //目前所在第几天
}

type RailwayService interface {
	SearchDirectly(departureStation, arrivalStation, speedOption string, sortOption int) (map[string][]dao.RailWay, error)
	SearchDirectlyOnline(departureStation, arrivalStation string) (map[string][]dao.RailWay, error)
	SearchWithOneTrans(departureStation, arrivalStation, speedOption string, sortOption int, limitStopTime, getAllResult int64) (map[string][]dao.RailWay, error)
	SearchWithOneSpecificTrans(departureStation, midStation, arrivalStation, speedOption string, sortOption int, limitStopTime int64) (map[string][]dao.RailWay, error)
	SearchWithTwoTrans(departureStation, arrivalStation, speedOption string, maxTrans, recordNumber int64, sortOption int) (map[string][]dao.RailWay, error)
}

type RailWayServiceImpl struct {
	RailWayDAO dao.RailWayDAO
	StationDAO dao.StationDAO
}

var (
	RailWayDAO dao.RailWayDAO
	R          RailWayServiceImpl
	_          RailwayService = (*RailWayServiceImpl)(nil)

	//图，前面的string是图中的点，以D或者A开头（表示出发还是到达）加上站点名加上车次NO加上第几天的车；后面的[]是从这个点出发的边 在站内转乘时TrainNumber记为Waiting，TrainNo为arrival的TrainNo，这样能够找到下一班车所在点
	Graph               = make(map[string][]dao.RailWay)
	TemplateGraph       = make(map[string][]dao.RailWay) //记录临时添加的点和边，后续需要删除
	KeyStationDeparture = make(map[string][]dao.RailWay) //记录关键站点的所有离开的车
	KeyStationArrival   = make(map[string][]dao.RailWay) //记录关键站点的所有到达的车
	dist                = make([]map[string]AnalyseTrans, 0)
)

func NewRailwayService(RailWayDAO dao.RailWayDAO, StationDAO dao.StationDAO) RailWayServiceImpl {
	return RailWayServiceImpl{
		RailWayDAO: RailWayDAO,
		StationDAO: StationDAO,
	}
}

func (r *RailWayServiceImpl) SearchDirectly(departureStation, arrivalStation, speedOption string, sortOption int) (returnResult map[string][]dao.RailWay, err error) {
	if !r.checkStation(departureStation) || !r.checkStation(arrivalStation) {
		log.Printf("[SearchDirectly] stationNotFind")
		return nil, errors.New("stationNotFind")
	}
	result := make([]dao.RailWay, 0)
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
	case LowPriceFirst:
		result = sortByLowPriceFirst(result)
	default:
		result = sortByEarlyFirst(result)
	}
	return turnSliceToMap(result), nil
}
func (r *RailWayServiceImpl) SearchDirectlyOnline(departureStation, arrivalStation string) (map[string][]dao.RailWay, error) {
	return nil, errors.New("not implement")
}

func (r *RailWayServiceImpl) SearchWithOneSpecificTrans(departureStation, midStation, arrivalStation, speedOption string, sortOption int, limitStopTime int64) (map[string][]dao.RailWay, error) {
	if !r.checkStation(departureStation) || !r.checkStation(arrivalStation) || !r.checkStation(midStation) {
		log.Printf("[SearchWithOneSpecificTrans] stationNotFind")
		return nil, errors.New("stationNotFind")
	}
	departTrain, err := r.RailWayDAO.GetRailWayByDepartureStationAndArrivalStation(departureStation, midStation)
	if err != nil {
		log.Printf("[SearchWithOneSpecificTrans] err:%s", err.Error())
		return nil, err
	}
	arrivalTrain, err := r.RailWayDAO.GetRailWayByDepartureStationAndArrivalStation(midStation, arrivalStation)
	if err != nil {
		log.Printf("[SearchWithOneSpecificTrans] err:%s", err.Error())
		return nil, err
	}
	result := CombineTrainSchedule(departTrain, arrivalTrain, speedOption)
	return SortTransResult(result, sortOption, limitStopTime, 0), nil
}

func (r *RailWayServiceImpl) SearchWithOneTrans(departureStation, arrivalStation, speedOption string, sortOption int, limitStopTime, getAllResult int64) (map[string][]dao.RailWay, error) {
	if !r.checkStation(departureStation) || !r.checkStation(arrivalStation) {
		log.Printf("[SearchWithOneTrans] stationNotFind")
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

func (r *RailWayServiceImpl) SearchWithTwoTrans(departureStation, arrivalStation, speedOption string, maxTrans, recordNumber int64, sortOption int) (map[string][]dao.RailWay, error) {
	if !r.checkStation(departureStation) || !r.checkStation(arrivalStation) {
		log.Printf("[SearchWithTwoTrans] stationNotFind")
		return nil, errors.New("stationNotFind")
	}
	TemplateGraph = make(map[string][]dao.RailWay)
	forbidTrain := make([]string, 0)
	answer := make(map[string][]dao.RailWay)
	err := r.AddNewStation(departureStation, true, 0)
	if err != nil {
		log.Printf("[SearchWithTwoTrans ] err:%s", err.Error())
		return nil, err
	}
	err = r.AddNewStation(arrivalStation, false, 0)
	if err != nil {
		log.Printf("[SearchWithTwoTrans ] err:%s", err.Error())
		return nil, err
	}
	for i := int64(0); i < recordNumber; i++ {
		result := Dijkstra(departureStation, arrivalStation, speedOption, forbidTrain, maxTrans, sortOption)
		if result.AllRunningTime > 1440*30 {
			break
		}
		title, railways := r.convertAnalyseToRailways(result)
		answer[title] = railways
		forbidTrain = append(forbidTrain, result.NowTrainNo)
	}
	return answer, nil
}

func (r *RailWayServiceImpl) convertAnalyseToRailways(trans AnalyseTrans) (string, []dao.RailWay) {
	title := ""
	result := make([]dao.RailWay, 0)
	for index, trainNumber := range trans.TrainNumber {
		title = title + trainNumber + "/"
		departureStation := trans.StationSequence[index]
		arrivalStation := ""
		if index != len(trans.StationSequence)-1 {
			arrivalStation = trans.StationSequence[index+1]
		} else {
			arrivalStation = trans.NowStation
		}
		train, err := r.RailWayDAO.GetRailWayByDepartureStationAndArrivalStationAndTrainNo(departureStation, arrivalStation, trans.TrainNo[index])
		if err != nil {
			fmt.Println("[convertAnalyseToRailways] GetRailWayByDepartureStationAndArrivalStationAndTrainNo Error")
			return "", []dao.RailWay{}
		}
		if train == nil {
			fmt.Println("[convertAnalyseToRailways] train nil!")
			return "", []dao.RailWay{}
		}
		if train.DepartureStation != departureStation {
			fmt.Println("[convertAnalyseToRailways] train different departure station!")
			fmt.Println(*train)
		}
		result = append(result, *train)
	}
	title = title + strconv.FormatInt(trans.AllRunningTime, 10)
	return title, result
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

func turnSliceToMap(Railways []dao.RailWay) map[string][]dao.RailWay {
	results := make(map[string][]dao.RailWay)
	for _, Railway := range Railways {
		runningTime, _ := GetTime(Railway.RunningTime)
		index := Railway.TrainNumber + "/" + strconv.FormatInt(runningTime, 10)
		results[index] = []dao.RailWay{Railway}
	}
	return results
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
func sortByLowPriceFirst(result []dao.RailWay) []dao.RailWay {
	sort.Slice(result, func(i, j int) bool {
		if result[i].Price != result[j].Price {
			return result[i].Price < result[j].Price
		}
		iDepartTime, _ := GetTime(result[i].DepartureTime)
		jDepartTime, _ := GetTime(result[j].DepartureTime)
		return iDepartTime < jDepartTime
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

func CombineTrainSchedule(departTrain, arrivalTrain []dao.RailWay, speedOption string) map[string][]dao.RailWay {
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
	file, err := excelize.OpenFile("train_ticket_prices_2.xlsx")
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
	rememberTrainNo := make(map[string]string)
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
		originalRailway := dao.RailWay{
			TrainNumber:      row[0],
			TrainNo:          row[1],
			DepartureStation: row[2],
			ArrivalStation:   row[3],
			DepartureTime:    row[4],
			ArrivalTime:      row[5],
			RunningTime:      row[6],
			YWPrice:          GetPrice(row[7]),
			YZPrice:          GetPrice(row[8]),
			RWPrice:          GetPrice(row[9]),
			ZEPrice:          GetPrice(row[10]),
			ZYPrice:          GetPrice(row[11]),
			SWZPrice:         GetPrice(row[12]),
			TZPrice:          GetPrice(row[13]),
			GRPrice:          GetPrice(row[14]),
		}
		_, ok := rememberTrainNo[originalRailway.TrainNo+originalRailway.DepartureStation+originalRailway.ArrivalStation]
		if ok {
			continue
		}
		rememberTrainNo[originalRailway.TrainNo+originalRailway.DepartureStation+originalRailway.ArrivalStation] = originalRailway.TrainNo
		originalRailway.Price = GetLowPrice(originalRailway)
		railWays = append(railWays, originalRailway)
		if len(railWays) > 50 {
			err = RailWayDAO.BatchCreateRailWays(railWays)
			if err != nil {
				return err
			}
			sum = sum + 1
			railWays = make([]dao.RailWay, 0)
			if sum%200 == 0 {
				fmt.Println(sum)
			}
		}
	}
	err = RailWayDAO.BatchCreateRailWays(railWays)
	if err != nil {
		return err
	}
	fmt.Println("railway create success")
	return nil
}
func GetPrice(price string) float64 {
	fPrice, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return 0
	}
	return fPrice / 10
}

func GetLowPrice(railway dao.RailWay) float64 {
	price := float64(10000)
	price = comparePrice(railway.YWPrice, price)
	price = comparePrice(railway.YZPrice, price)
	price = comparePrice(railway.RWPrice, price)
	price = comparePrice(railway.ZEPrice, price)
	price = comparePrice(railway.ZYPrice, price)
	price = comparePrice(railway.SWZPrice, price)
	price = comparePrice(railway.TZPrice, price)
	price = comparePrice(railway.GRPrice, price)
	return price
}

func comparePrice(priceA, priceB float64) float64 {
	if priceA < 0.5 {
		return priceB
	}
	if priceA < priceB {
		return priceA
	}
	return priceB
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
