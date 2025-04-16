package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"railway/dao"
	"railway/service"
	"sort"
	"strconv"
	"strings"
)

type HandlerImpl struct {
	RailWayServiceImpl service.RailWayServiceImpl
}

var (
	_ Handler = (*HandlerImpl)(nil)
	H HandlerImpl
)

func NewHandler(RailWayServiceImpl service.RailWayServiceImpl) HandlerImpl {
	return HandlerImpl{
		RailWayServiceImpl: RailWayServiceImpl,
	}
}

type RequestStation struct {
	Keyword string `json:"keyword"` // JSON 标签，表示 JSON 中的字段名为 "keyword"
}
type ResponseStation struct {
	Results []string `json:"results"` // 返回结果数组，类型为字符串数组
}

type RequestSearch struct {
	From        string   `json:"from"`
	To          string   `json:"to"`
	SortBy      int64    `json:"sort_by"`
	MaxTransfer string   `json:"max_transfer"`
	MidStations []string `json:"midStations"`
	TrainType   string   `json:"train_type"`
}

type ResponseSearch struct {
	Index         string        `json:"index"`
	TotalTime     int64         `json:"total_time"`
	TotalPrice    float64       `json:"total_price"`
	DepartureTime string        `json:"start_time"`
	Railway       []dao.RailWay `json:"railway"`
}

func StartNgork() {
	r := gin.Default()

	// 定义一个 GET 请求接口
	r.GET("/test", func(c *gin.Context) {
		// 返回一个字符串表示成功连接
		c.String(http.StatusOK, "连接成功")
	})
	r.POST("/station", H.stationHandler)
	r.POST("/search", H.searchHandler)
	// 启动 HTTPS 服务
	err := r.RunTLS(":443", "cert.pem", "server.key")
	if err != nil {
		return
	}
}

type Handler interface {
	stationHandler(c *gin.Context)
	searchHandler(c *gin.Context)
}

func (h *HandlerImpl) stationHandler(c *gin.Context) {
	var req RequestStation
	// 解析请求体中的 JSON 数据
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果解析失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		fmt.Println(err)
	}

	// 调用服务层（例如 RailWayDAO）来获取查询结果
	resultCities, err := h.RailWayServiceImpl.StationDAO.GetCityByPrefixName(req.Keyword)
	if err != nil {
		// 如果查询出错，返回 500 错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching results"})
	}
	resultStations, err := h.RailWayServiceImpl.StationDAO.GetStationByPrefixName(req.Keyword)
	if err != nil {
		// 如果查询出错，返回 500 错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching results"})
	}
	results := make([]string, 0)
	resultStation := make([]string, 0)
	resultCity := make([]string, 0)
	city := make(map[string]string)
	for _, result := range resultStations {
		resultStation = append(resultStation, result.StationName)
	}
	for _, result := range resultCities {
		_, ok := city[result.CityName]
		if !ok {
			city[result.CityName] = result.CityName
			resultCity = append(resultCity, result.CityName+"（市）")
		}
	}
	results = append(results, resultCity...)
	results = append(results, resultStation...)

	// 返回查询结果
	c.JSON(http.StatusOK, ResponseStation{Results: results})
}

func (h *HandlerImpl) searchHandler(c *gin.Context) {
	var req RequestSearch
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		fmt.Println(c)
	}
	maxTransfer, err := strconv.ParseInt(req.MaxTransfer, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		fmt.Println(c)
	}
	req.SortBy = req.SortBy + 1
	if req.SortBy > 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
	}
	results := make(map[string][]dao.RailWay)
	departStations, err := h.getStations(req.From)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getStations fetching results"})
	}
	arrivalStations, err := h.getStations(req.To)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getStations fetching results"})
	}
	if len(req.MidStations) > 0 {
		midStations, err := h.getStations(req.MidStations[0])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getStations fetching results"})
		}
		if len(midStations) > 0 {
			for _, midStation := range midStations {
				for _, departStation := range departStations {
					for _, arrivalStation := range arrivalStations {
						templateResults, err := h.searchWithStations(departStation, midStation, arrivalStation, req.TrainType, int(req.SortBy), maxTransfer)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "Error searchWithStations fetching results"})
						}
						results = combineMap(results, templateResults)
					}
				}
			}
		}
	} else {
		for _, departStation := range departStations {
			for _, arrivalStation := range arrivalStations {
				templateResults, err := h.searchWithStations(departStation, "", arrivalStation, req.TrainType, int(req.SortBy), maxTransfer)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error searchWithStations fetching results"})
				}
				results = combineMap(results, templateResults)
			}
		}
	}
	returnResult := turnMapToResponseSlice(results)
	switch req.SortBy {
	case service.LowRunningTimeFirst:
		returnResult = sortTemplateStructByLowRunningTime(returnResult)
	case service.HighRunningTimeFirst:
		returnResult = sortTemplateStructByHighRunningTime(returnResult)
	case service.EarlyFirst:
		returnResult = sortTemplateStructByEarlyFirst(returnResult)
	case service.LateFirst:
		returnResult = sortTemplateStructByLateFirst(returnResult)
	case service.LowPriceFirst:
		returnResult = sortTemplateStructByLowPrice(returnResult)
	case service.HighPriceFirst:
		returnResult = sortTemplateStructByHighPrice(returnResult)
	default:
		returnResult = sortTemplateStructByLowRunningTime(returnResult)
	}
	c.JSON(http.StatusOK, returnResult)
}

func (h *HandlerImpl) searchWithStations(departureStation, midStation, arrivalStation, speedOption string, sortOption int, maxTrans int64) (map[string][]dao.RailWay, error) {
	results := make(map[string][]dao.RailWay)
	if len(midStation) > 0 {
		templateResult, err := h.RailWayServiceImpl.SearchWithOneSpecificTrans(departureStation, midStation, arrivalStation, speedOption, sortOption, service.DefaultStopTime)
		if err != nil {
			return nil, err
		}
		results = templateResult
	} else {
		templateResult, err := h.RailWayServiceImpl.SearchDirectly(departureStation, arrivalStation, speedOption, sortOption)
		if err != nil {
			return nil, err
		}
		results = combineMap(results, templateResult)

		if maxTrans >= 1 {
			templateResult, err = h.RailWayServiceImpl.SearchWithOneTrans(departureStation, arrivalStation, speedOption, sortOption, service.DefaultStopTime, 0)
			if err != nil {
				return nil, err
			}
			results = combineMap(results, templateResult)
		}
		if maxTrans >= 2 && (sortOption == service.LowRunningTimeFirst || sortOption == service.LowPriceFirst) {
			templateResult, err = h.RailWayServiceImpl.SearchWithTwoTrans(departureStation, arrivalStation, speedOption, maxTrans+1, service.DefaultResultNumber, sortOption)
			if err != nil {
				return nil, err
			}
			results = combineMap(results, templateResult)
		}
	}
	return results, nil
}

func (h *HandlerImpl) getStations(inputStation string) ([]string, error) {
	if strings.Contains(inputStation, "（市）") {
		inputCity := strings.TrimSuffix(inputStation, "（市）")
		startStations, err := h.RailWayServiceImpl.StationDAO.GetStationByPrefixName(inputCity)
		if err != nil {
			return nil, err
		}
		results := make([]string, 0)
		for _, station := range startStations {
			results = append(results, station.StationName)
		}
		return results, nil
	} else {
		return []string{inputStation}, nil
	}
}

func combineMap(mapA, mapB map[string][]dao.RailWay) map[string][]dao.RailWay {
	for key, value := range mapB {
		mapA[key] = value
	}
	return mapA
}

func turnMapToResponseSlice(results map[string][]dao.RailWay) []ResponseSearch {
	returnResults := make([]ResponseSearch, 0)
	for key, value := range results {
		Price := float64(0)
		keyStrings := strings.Split(key, "/")
		TotalTime, _ := strconv.ParseInt(keyStrings[len(keyStrings)-1], 10, 64)
		for _, element := range value {
			Price = Price + element.Price
		}
		newReturnResult := ResponseSearch{
			Index:         key,
			TotalTime:     TotalTime,
			TotalPrice:    Price,
			DepartureTime: value[0].DepartureTime,
			Railway:       value,
		}
		returnResults = append(returnResults, newReturnResult)
	}
	return returnResults
}

func sortTemplateStructByLowRunningTime(result []ResponseSearch) []ResponseSearch {
	sort.Slice(result, func(i, j int) bool {
		if result[i].TotalTime == result[j].TotalTime {
			iDepartTime, _ := service.GetTime(result[i].Railway[0].DepartureTime)
			jDepartTime, _ := service.GetTime(result[j].Railway[0].DepartureTime)
			return iDepartTime < jDepartTime
		}
		return result[i].TotalTime < result[j].TotalTime
	})
	return result
}

func sortTemplateStructByHighRunningTime(result []ResponseSearch) []ResponseSearch {
	sort.Slice(result, func(i, j int) bool {
		if result[i].TotalTime == result[j].TotalTime {
			iDepartTime, _ := service.GetTime(result[i].Railway[0].DepartureTime)
			jDepartTime, _ := service.GetTime(result[j].Railway[0].DepartureTime)
			return iDepartTime < jDepartTime
		}
		return result[i].TotalTime > result[j].TotalTime
	})
	return result
}

func sortTemplateStructByEarlyFirst(result []ResponseSearch) []ResponseSearch {
	sort.Slice(result, func(i, j int) bool {
		iDepartTime, _ := service.GetTime(result[i].Railway[0].DepartureTime)
		jDepartTime, _ := service.GetTime(result[j].Railway[0].DepartureTime)
		if iDepartTime == jDepartTime {
			return result[i].TotalTime < result[j].TotalTime
		}
		return iDepartTime < jDepartTime
	})
	return result
}

func sortTemplateStructByLateFirst(result []ResponseSearch) []ResponseSearch {
	sort.Slice(result, func(i, j int) bool {
		iDepartTime, _ := service.GetTime(result[i].Railway[0].DepartureTime)
		jDepartTime, _ := service.GetTime(result[j].Railway[0].DepartureTime)
		if iDepartTime == jDepartTime {
			return result[i].TotalTime < result[j].TotalTime
		}
		return iDepartTime > jDepartTime
	})
	return result
}
func sortTemplateStructByLowPrice(result []ResponseSearch) []ResponseSearch {
	sort.Slice(result, func(i, j int) bool {
		if result[i].TotalPrice != result[j].TotalPrice {
			return result[i].TotalPrice < result[j].TotalPrice
		}
		iDepartTime, _ := service.GetTime(result[i].Railway[0].DepartureTime)
		jDepartTime, _ := service.GetTime(result[j].Railway[0].DepartureTime)
		if iDepartTime == jDepartTime {
			return result[i].TotalTime < result[j].TotalTime
		}
		return iDepartTime > jDepartTime
	})
	return result
}
func sortTemplateStructByHighPrice(result []ResponseSearch) []ResponseSearch {
	sort.Slice(result, func(i, j int) bool {
		if result[i].TotalPrice != result[j].TotalPrice {
			return result[i].TotalPrice > result[j].TotalPrice
		}
		iDepartTime, _ := service.GetTime(result[i].Railway[0].DepartureTime)
		jDepartTime, _ := service.GetTime(result[j].Railway[0].DepartureTime)
		if iDepartTime == jDepartTime {
			return result[i].TotalTime < result[j].TotalTime
		}
		return iDepartTime > jDepartTime
	})
	return result
}
