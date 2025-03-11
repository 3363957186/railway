package dao

import "gorm.io/gorm"

type RailWay struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	TrainNumber      string `gorm:"size:20" json:"train_number"`
	TrainNo          string `gorm:"size:20" json:"train_no"`
	DepartureStation string `gorm:"size:20" json:"departure_station"`
	DepartureTime    string `gorm:"size:20" json:"departure_time"`
	ArrivalStation   string `gorm:"size:20" json:"arrival_station"`
	ArrivalTime      string `gorm:"size:20" json:"arrival_time"`
	RunningTime      string `gorm:"size:20" json:"running_time"`
	ArrivalDay       uint
}

type RailWayDAO interface {
	CreateRailWay(railway *RailWay) error
	BatchCreateRailWays(railways []*RailWay) error
	GetRailWayByID(id int) (*RailWay, error)
	GetRailWayByTrainNumber(trainNumber string) (*RailWay, error)
	GetRailWayByDepartureStation(name string) ([]RailWay, error)
	GetRailWayByArrivalStation(name string) ([]RailWay, error)
	GetRailWayByDepartureStationAndArrivalStation(departureName, arrivalName string) ([]RailWay, error)
	GetAllRailWays() ([]RailWay, error)
	UpdateRailWays(station *RailWay) error
	DeleteRailWays(id int) error
}

type RailWayDAOImpl struct {
	DB *gorm.DB
}

func (RailWay) TableName() string {
	return "railway"
}

func NewRailWayDAO(db *gorm.DB) RailWayDAO {
	return &RailWayDAOImpl{
		DB: db,
	}
}

var _ RailWayDAO = (*RailWayDAOImpl)(nil)

func (dao *RailWayDAOImpl) CreateRailWay(railWay *RailWay) error {
	return dao.DB.Create(railWay).Error
}

func (dao *RailWayDAOImpl) BatchCreateRailWays(railways []*RailWay) error {
	if len(railways) == 0 {
		return nil
	}
	batchSize := 100
	return dao.DB.CreateInBatches(&railways, batchSize).Error
}

func (dao *RailWayDAOImpl) GetRailWayByID(id int) (*RailWay, error) {
	var railWay RailWay
	result := dao.DB.First(&railWay, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &railWay, nil
}

func (dao *RailWayDAOImpl) GetRailWayByTrainNumber(trainNumber string) (*RailWay, error) {
	var railWay RailWay
	result := dao.DB.Where("train_number = ?", trainNumber).First(&railWay)
	if result.Error != nil {
		return nil, result.Error
	}
	return &railWay, nil
}

func (dao *RailWayDAOImpl) GetRailWayByDepartureStation(name string) ([]RailWay, error) {
	railWays := make([]RailWay, 0)
	result := dao.DB.Where("departure_station = ?", name).First(&railWays)
	if result.Error != nil {
		return nil, result.Error
	}
	return railWays, nil
}

func (dao *RailWayDAOImpl) GetRailWayByArrivalStation(name string) ([]RailWay, error) {
	railWays := make([]RailWay, 0)
	result := dao.DB.Where("arrival_station = ?", name).Find(&railWays)
	if result.Error != nil {
		return nil, result.Error
	}
	return railWays, nil
}

func (dao *RailWayDAOImpl) GetRailWayByDepartureStationAndArrivalStation(departureName, arrivalName string) ([]RailWay, error) {
	railWays := make([]RailWay, 0)
	result := dao.DB.Where("departure_station = ? and arrival_name = ?", departureName, arrivalName).Find(&railWays)
	if result.Error != nil {
		return nil, result.Error
	}
	return railWays, nil
}

func (dao *RailWayDAOImpl) GetAllRailWays() ([]RailWay, error) {
	railWays := make([]RailWay, 0)
	result := dao.DB.Find(&railWays)
	if result.Error != nil {
		return nil, result.Error
	}
	return railWays, nil
}

func (dao *RailWayDAOImpl) UpdateRailWays(railWay *RailWay) error {
	return dao.DB.Save(railWay).Error
}

func (dao *RailWayDAOImpl) DeleteRailWays(id int) error {
	return dao.DB.Delete(&RailWay{}, id).Error
}
