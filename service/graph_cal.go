package service

import (
	"container/heap"
	"fmt"
	"log"
	"math"
	"railway/dao"
	"strconv"
	"strings"
)

func (r *RailWayServiceImpl) AddNewStation(stationName string, isDeparture bool, startTime int64) error {
	if isDeparture {
		departureTrains, err := r.RailWayDAO.GetRailWayByDepartureStation(stationName)
		if err != nil {
			log.Fatal(err)
			return err
		}
		_, isKey := KeyStation[stationName]
		departureTrains = getOneKeyTrains(departureTrains, false, 0, false, isKey)
		for _, train := range departureTrains {
			dTime, _ := GetTime(train.DepartureTime)
			if startTime <= dTime {
				value, ok := TemplateGraph[StartIndex]
				if ok {
					value = append(value, train)
					TemplateGraph[StartIndex] = value
				} else {
					TemplateGraph[StartIndex] = []dao.RailWay{train}
				}
				AddStationTrans(train, KeyStationDeparture[train.ArrivalStation], DefaultStopTime)
			}
		}
	} else {
		_, ok := KeyStation[stationName]
		if ok {
			return nil
		}
		arrivalTrains, err := r.RailWayDAO.GetRailWayByArrivalStation(stationName)
		if err != nil {
			log.Fatal(err)
			return err
		}
		_, isKey := KeyStation[stationName]
		arrivalTrains = getOneKeyTrains(arrivalTrains, false, 0, false, isKey)
		getOneKeyTrains(arrivalTrains, true, 0, true, isKey)
		getOneKeyTrains(arrivalTrains, true, 1, true, isKey)
		getOneKeyTrains(arrivalTrains, true, 2, true, isKey)
		if !isKey {
			for _, train := range arrivalTrains {
				for _, arrivalTrain := range KeyStationArrival[train.DepartureStation] {
					turnADToEdges(arrivalTrain, train, 2, DefaultStopTime, true)
				}
			}
		}
	}
	return nil
}

func AddStationTrans(arriveTrain dao.RailWay, departureKeyTrains []dao.RailWay, limitStopTime int64) {
	if len(departureKeyTrains) == 0 {
		return
	}
	for _, train := range departureKeyTrains {
		if train.TrainNo == arriveTrain.TrainNo {
			turnADToEdges(arriveTrain, train, 2, 0, true)
		}
	}
	for _, train := range departureKeyTrains {
		aTime, _ := GetTime(arriveTrain.ArrivalTime)
		dTime, _ := GetTime(train.DepartureTime)
		if aTime+limitStopTime < dTime && train.TrainNo != arriveTrain.TrainNo {
			turnADToEdges(arriveTrain, train, 2, limitStopTime, true)
			return
		}
	}
	//如果当天没有可以换乘的车则搜寻第二天的
	for _, train := range departureKeyTrains {
		aTime, _ := GetTime(arriveTrain.ArrivalTime)
		dTime, _ := GetTime(train.DepartureTime)
		if aTime+limitStopTime < dTime+1440 && train.TrainNo != arriveTrain.TrainNo {
			turnADToEdges(arriveTrain, train, 2, limitStopTime, true)
			return
		}
	}
}

func (r *RailWayServiceImpl) DeleteNewStation(stationName string, isDeparture bool) {
	if isDeparture {
		Graph[StartIndex] = []dao.RailWay{}
	}
	return
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
		KeyStationDeparture[key] = departureTrains
		KeyStationArrival[key] = arrivalTrains

		length := len(departureTrains)
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
func getOneKeyTrains(input []dao.RailWay, isAddGraph bool, dayTime int, IsTemplate, isKey bool) []dao.RailWay {
	result := make([]dao.RailWay, 0)
	rememberTrainNo := make(map[string]string)
	for _, v := range input {
		_, ok := rememberTrainNo[v.TrainNo]
		if ok {
			continue
		}
		ok = false
		if isKey {
			if checkKeyStation(v.DepartureStation) && checkKeyStation(v.ArrivalStation) {
				ok = true
			}
		} else {
			if checkKeyStation(v.DepartureStation) || checkKeyStation(v.ArrivalStation) {
				ok = true
			}
		}
		if ok {
			rememberTrainNo[v.TrainNo] = v.TrainNo
			result = append(result, v)
			if isAddGraph {
				//加边
				vv := v
				vv.ArrivalDay = vv.ArrivalDay + uint(dayTime)
				departIndex := "D/" + v.DepartureStation + "/" + v.TrainNo + "/" + strconv.Itoa(dayTime)
				arriveIndex := "A/" + v.ArrivalStation + "/" + v.TrainNo + "/" + strconv.Itoa(dayTime)

				if IsTemplate {
					value, ok := TemplateGraph[departIndex]
					if ok {
						value = append(value, vv)
						TemplateGraph[departIndex] = value
					} else {
						TemplateGraph[departIndex] = []dao.RailWay{vv}
					}
					_, ok = TemplateGraph[arriveIndex]
					if !ok {
						TemplateGraph[arriveIndex] = []dao.RailWay{}
					}
				} else {
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
	}
	return result
}

func buildDepartureWaitingEdges(arrival, departure dao.RailWay, maxArrivalDay int64) {
	for arrivalDay := int64(0); arrivalDay <= maxArrivalDay; arrivalDay++ {
		newEdge := dao.RailWay{
			TrainNumber:      Waiting,
			TrainNo:          arrival.TrainNo,
			DepartureStation: departure.DepartureStation,
			ArrivalStation:   arrival.DepartureStation,
			DepartureTime:    departure.DepartureTime,
			ArrivalTime:      arrival.DepartureTime,
			ArrivalDay:       uint(arrivalDay),
		}
		if newEdge.ArrivalStation != newEdge.DepartureStation {
			fmt.Println("[buildDepartureWaitingEdges] WRONG!!!")
			fmt.Println(arrival, departure)
			return
		}
		runningTime := CalculateStopTime(newEdge.DepartureTime, newEdge.ArrivalTime)
		newEdge.RunningTime = TurnToTime(runningTime)
		dTime, _ := GetTime(newEdge.DepartureTime)
		aTime, _ := GetTime(newEdge.ArrivalTime)
		if dTime > aTime {
			newEdge.ArrivalDay = newEdge.ArrivalDay + 1
			arrivalDay = arrivalDay + 1
		}
		departIndex := "D/" + newEdge.DepartureStation + "/" + departure.TrainNo + "/" + strconv.FormatInt(arrivalDay, 10)
		value, ok := Graph[departIndex]
		if ok {
			value = append(value, newEdge)
			Graph[departIndex] = value
		} else {
			Graph[departIndex] = []dao.RailWay{newEdge}
		}
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
				turnADToEdges(arrival, departure, 2, 0, false)
			}
		}
	}
	for _, arrival := range arrivalTrains {
		isSuccess := false
		for i := dIndex; i < dLength; i++ {
			aTime, _ := GetTime(arrival.ArrivalTime)
			dTime, _ := GetTime(departureTrains[dIndex].DepartureTime)
			if aTime+limitStopTime <= dTime && arrival.TrainNo != departureTrains[dIndex].TrainNo {
				turnADToEdges(arrival, departureTrains[dIndex], 2, limitStopTime, false)
				isSuccess = true
				dIndex = i
				break
			}
			dIndex = i
		}
		if !isSuccess {
			for _, train := range departureTrains {
				aTime, _ := GetTime(arrival.ArrivalTime)
				dTime, _ := GetTime(train.DepartureTime)
				if aTime+limitStopTime <= dTime+1440 && arrival.TrainNo != train.TrainNo {
					turnADToEdges(arrival, train, 2, limitStopTime, false)
					isSuccess = true
				}
			}
		}
	}
}

func turnADToEdges(arrival, departure dao.RailWay, maxArrivalDay, limitStopTime int64, isTemplate bool) {
	for arrivalDay := int64(0); arrivalDay <= maxArrivalDay; arrivalDay++ {
		templatearrivalDay := arrivalDay
		newEdge := dao.RailWay{
			TrainNumber:      Waiting,
			TrainNo:          departure.TrainNo,
			DepartureStation: arrival.ArrivalStation,
			ArrivalStation:   departure.DepartureStation,
			DepartureTime:    arrival.ArrivalTime,
			ArrivalTime:      departure.DepartureTime,
			ArrivalDay:       uint(arrivalDay),
		}
		if newEdge.ArrivalStation != newEdge.DepartureStation {
			fmt.Println("[turnADToEdges] WRONG!!!")
			fmt.Println(arrival, departure)
			return
		}
		runningTime := CalculateStopTime(newEdge.DepartureTime, newEdge.ArrivalTime)
		newEdge.RunningTime = TurnToTime(runningTime)
		dTime, _ := GetTime(newEdge.DepartureTime)
		aTime, _ := GetTime(newEdge.ArrivalTime)
		//同一班列车的limitStopTime要为0
		if dTime+limitStopTime > aTime {
			newEdge.ArrivalDay = newEdge.ArrivalDay + 1
			templatearrivalDay = arrivalDay + 1
		}
		arrivalIndex := "A/" + newEdge.DepartureStation + "/" + arrival.TrainNo + "/" + strconv.FormatInt(templatearrivalDay, 10)
		if isTemplate {
			value, ok := TemplateGraph[arrivalIndex]
			if ok {
				value = append(value, newEdge)
				TemplateGraph[arrivalIndex] = value
			} else {
				TemplateGraph[arrivalIndex] = []dao.RailWay{newEdge}
			}
		} else {
			value, ok := Graph[arrivalIndex]
			if ok {
				value = append(value, newEdge)
				Graph[arrivalIndex] = value
			} else {
				Graph[arrivalIndex] = []dao.RailWay{newEdge}
			}
		}
	}
}

func Dijkstra(startStation, endStation, speedOption string, forbidTrain []string, maxTrans int64, sortOptions int) AnalyseTrans {
	//for key, value := range Graph {
	//	stringIndex := strings.Split(key, "/")
	//	if len(stringIndex) > 2 && stringIndex[1] == "乌鲁木齐" {
	//		fmt.Println(value)
	//	}
	//}

	// 初始化最短路径映射
	dist = make([]map[string]AnalyseTrans, 0)
	for i := int64(0); i <= maxTrans; i++ {
		dist = append(dist, make(map[string]AnalyseTrans))
	}
	// 初始化所有点的路径值为最大
	for node := range Graph {
		for i := int64(0); i <= maxTrans; i++ {
			dist[i][node] = AnalyseTrans{
				AllRunningTime: math.MaxInt64,
				ToTalPrice:     math.MaxInt64,
				TransFerTimes:  math.MaxInt64,
			}
		}
	}
	dist[0][StartIndex] = AnalyseTrans{
		AllRunningTime:  0,
		TransFerTimes:   0,
		ToTalPrice:      0,
		NowStatus:       "D",
		TrainNumber:     []string{},
		TrainNo:         []string{},
		StationSequence: []string{},
	}

	// 初始化最小堆
	pq := &PriorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &Item{node: StartIndex, allTime: 0, transferTimes: 0})
	//fmt.Println(Graph[StartIndex])
	// 运行 Dijkstra
	for pq.Len() > 0 {
		curr := heap.Pop(pq).(*Item)
		currNode, currTime, currTransfers, currPrice := curr.node, curr.allTime, curr.transferTimes, curr.price
		// 如果当前路径已经不是最短路径，则跳过
		if currTime > dist[currTransfers][currNode].AllRunningTime ||
			(currTime == dist[currTransfers][currNode].AllRunningTime && currTransfers > dist[currTransfers][currNode].TransFerTimes) {
			continue
		}
		indexString := strings.Split(currNode, "/")
		if len(indexString) > 1 && indexString[1] == endStation {
			return dist[currTransfers][currNode]
		}
		//fmt.Println(dist[currTransfers][currNode])
		// 遍历邻接点
		edges, ok := TemplateGraph[currNode]
		if ok {
			for _, edge := range edges {
				//判断specialTag
				if curr.specialTag == true && edge.DepartureStation == edge.ArrivalStation {
					continue
				}
				if sortOptions == LowPriceFirst {
					item := getAnalyseTransByPrice(edge, forbidTrain, currNode, speedOption, currTransfers, currTime, maxTrans, currPrice)
					if item != nil {
						heap.Push(pq, item)
					}
				} else {
					item := getAnalyseTransByTime(edge, forbidTrain, currNode, speedOption, currTransfers, currTime, maxTrans, currPrice)
					if item != nil {
						heap.Push(pq, item)
					}
				}

			}
		}
		edges, ok = Graph[currNode]
		if ok {
			for _, edge := range edges {
				if curr.specialTag == true && edge.DepartureStation == edge.ArrivalStation {
					continue
				}
				if sortOptions == LowPriceFirst {
					item := getAnalyseTransByPrice(edge, forbidTrain, currNode, speedOption, currTransfers, currTime, maxTrans, currPrice)
					if item != nil {
						heap.Push(pq, item)
					}
				} else {
					item := getAnalyseTransByTime(edge, forbidTrain, currNode, speedOption, currTransfers, currTime, maxTrans, currPrice)
					if item != nil {
						heap.Push(pq, item)
					}
				}
			}
		}
	}
	return AnalyseTrans{
		AllRunningTime: math.MaxInt64,
		TransFerTimes:  math.MaxInt64,
	}
}

func isInForbid(trainNo string, forbidTrains []string) bool {
	if len(forbidTrains) == 0 {
		return false
	}
	for _, forbidTrain := range forbidTrains {
		if trainNo == forbidTrain {
			return true
		}
	}
	return false
}

// 最短路的具体实现
// 转乘的逻辑是如果当前边是出发边且不是站内Waiting边且和点本身的TrainNo不一致，那么将视为进行转乘，并且将列车信息写入Dist当中
func getAnalyseTransByTime(edge dao.RailWay, forbidTrain []string, currNode, speedOption string, currTransfers, currTime, maxTrans int64, currPrice float64) *Item {
	if isInForbid(edge.TrainNo, forbidTrain) {
		return nil
	}
	if speedOption == OnlyHighSpeed && edge.IsHighSpeed == 0 {
		return nil
	}
	if speedOption == OnlyLowSpeed && edge.IsHighSpeed == 1 {
		return nil
	}
	//超过三天的行程不记录
	if edge.ArrivalDay > 2 {
		return nil
	}
	var (
		nextNode   string
		status     string
		travelTime int64
		transfers  int64
		specialTag bool
	)

	if edge.TrainNumber == Waiting {
		nextNode = "D/" + edge.ArrivalStation + "/" + edge.TrainNo + "/" + strconv.Itoa(int(edge.ArrivalDay))
		status = "D"
	} else {
		nextNode = "A/" + edge.ArrivalStation + "/" + edge.TrainNo + "/" + strconv.Itoa(int(edge.ArrivalDay))
		status = "A"
	}

	travelTime, _ = GetTime(edge.RunningTime)
	length := len(dist[currTransfers][currNode].TrainNo)
	if dist[currTransfers][currNode].NowStatus == "D" && edge.TrainNumber != Waiting && (length == 0 || dist[currTransfers][currNode].TrainNo[length-1] != edge.TrainNo) {
		transfers = 1
	} else {
		transfers = 0
	}
	//增加标签判断
	if dist[currTransfers][currNode].NowStatus == "A" && travelTime < DefaultStopTime {
		specialTag = true
	} else {
		specialTag = false
	}

	newTime := currTime + travelTime
	newTransfers := currTransfers + transfers
	if newTransfers > maxTrans {
		return nil
	}
	// 如果找到更优路径，则更新
	_, ok := dist[newTransfers][nextNode]
	if !ok {
		dist[newTransfers][nextNode] = AnalyseTrans{
			AllRunningTime: math.MaxInt64,
			TransFerTimes:  math.MaxInt64,
			ToTalPrice:     math.MaxInt64,
		}
	}
	if newTime < dist[newTransfers][nextNode].AllRunningTime ||
		(newTime == dist[newTransfers][nextNode].AllRunningTime && newTransfers < dist[newTransfers][nextNode].TransFerTimes) {
		newAnalyseTrans := AnalyseTrans{
			NowTrainNumber:  edge.TrainNumber,
			NowTrainNo:      edge.TrainNo,
			NowStation:      edge.ArrivalStation,
			NowStatus:       status,
			TrainNumber:     dist[currTransfers][currNode].TrainNumber,
			TrainNo:         dist[currTransfers][currNode].TrainNo,
			StationSequence: dist[currTransfers][currNode].StationSequence,
			AllRunningTime:  newTime,
			TransFerTimes:   newTransfers,
			ToTalPrice:      currPrice + edge.Price,
			NowArrivalDay:   int64(edge.ArrivalDay),
		}
		if transfers == 1 {
			newAnalyseTrans.TrainNumber = append(newAnalyseTrans.TrainNumber, edge.TrainNumber)
			newAnalyseTrans.TrainNo = append(newAnalyseTrans.TrainNo, edge.TrainNo)
			newAnalyseTrans.StationSequence = append(newAnalyseTrans.StationSequence, edge.DepartureStation)
		}
		dist[newTransfers][nextNode] = newAnalyseTrans

		return &Item{node: nextNode, allTime: newTime, transferTimes: newTransfers, specialTag: specialTag}
	}
	return nil
}

func getAnalyseTransByPrice(edge dao.RailWay, forbidTrain []string, currNode, speedOption string, currTransfers, currTime, maxTrans int64, currPrice float64) *Item {
	if isInForbid(edge.TrainNo, forbidTrain) {
		return nil
	}
	if speedOption == OnlyHighSpeed && edge.IsHighSpeed == 0 {
		return nil
	}
	if speedOption == OnlyLowSpeed && edge.IsHighSpeed == 1 {
		return nil
	}
	//超过三天的行程不记录
	if edge.ArrivalDay > 2 {
		return nil
	}
	var (
		nextNode   string
		status     string
		travelTime int64
		transfers  int64
	)

	if edge.TrainNumber == Waiting {
		nextNode = "D/" + edge.ArrivalStation + "/" + edge.TrainNo + "/" + strconv.Itoa(int(edge.ArrivalDay))
		status = "D"
	} else {
		nextNode = "A/" + edge.ArrivalStation + "/" + edge.TrainNo + "/" + strconv.Itoa(int(edge.ArrivalDay))
		status = "A"
	}

	travelTime, _ = GetTime(edge.RunningTime)
	length := len(dist[currTransfers][currNode].TrainNo)
	if dist[currTransfers][currNode].NowStatus == "D" && edge.TrainNumber != Waiting && (length == 0 || dist[currTransfers][currNode].TrainNo[length-1] != edge.TrainNo) {
		transfers = 1
	} else {
		transfers = 0
	}

	newTime := currTime + travelTime
	newTransfers := currTransfers + transfers
	newPrice := currPrice + edge.Price
	if newTransfers > maxTrans {
		return nil
	}
	// 如果找到更优路径，则更新
	_, ok := dist[newTransfers][nextNode]
	if !ok {
		dist[newTransfers][nextNode] = AnalyseTrans{
			AllRunningTime: math.MaxInt64,
			TransFerTimes:  math.MaxInt64,
			ToTalPrice:     math.MaxFloat64,
		}
	}
	if newPrice < dist[newTransfers][nextNode].ToTalPrice ||
		(newPrice == dist[newTransfers][nextNode].ToTalPrice && newTransfers < dist[newTransfers][nextNode].TransFerTimes) {
		newAnalyseTrans := AnalyseTrans{
			NowTrainNumber:  edge.TrainNumber,
			NowTrainNo:      edge.TrainNo,
			NowStation:      edge.ArrivalStation,
			NowStatus:       status,
			TrainNumber:     dist[currTransfers][currNode].TrainNumber,
			TrainNo:         dist[currTransfers][currNode].TrainNo,
			StationSequence: dist[currTransfers][currNode].StationSequence,
			AllRunningTime:  newTime,
			TransFerTimes:   newTransfers,
			ToTalPrice:      newPrice,
			NowArrivalDay:   int64(edge.ArrivalDay),
		}
		if transfers == 1 {
			newAnalyseTrans.TrainNumber = append(newAnalyseTrans.TrainNumber, edge.TrainNumber)
			newAnalyseTrans.TrainNo = append(newAnalyseTrans.TrainNo, edge.TrainNo)
			newAnalyseTrans.StationSequence = append(newAnalyseTrans.StationSequence, edge.DepartureStation)
		}
		dist[newTransfers][nextNode] = newAnalyseTrans

		return &Item{node: nextNode, allTime: newTime, transferTimes: newTransfers}
	}
	return nil
}
