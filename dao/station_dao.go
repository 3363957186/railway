package dao

import (
	"gorm.io/gorm"
)

// Station 车站数据模型
type Station struct {
	ID                 int    `gorm:"primaryKey;autoIncrement"`
	StationAbbr        string `gorm:"size:20" json:"station_abbr"`         // 车站简称
	StationName        string `gorm:"size:100" json:"station_name"`        // 车站名
	StationCode        string `gorm:"unique;size:20" json:"station_code"`  // 车站代号
	StationPinyin      string `gorm:"size:100" json:"station_pinyin"`      // 车站拼音
	StationFirstLetter string `gorm:"size:10" json:"station_first_letter"` // 车站首字母
	StationNumber      string `gorm:"size:20" json:"station_number"`       // 车站标号
	CityCode           string `gorm:"size:10" json:"city_code"`            // 城市代码
	CityName           string `gorm:"size:100" json:"city_name"`           // 车站所属城市
	IsKeyStation       int    ` json:"is_key_station"`                     //是否是二次转乘选择站点
}

type StationDAO interface {
	CreateStation(station *Station) error
	GetStationByID(id int) (*Station, error)
	GetStationByName(name string) (*Station, error)
	GetStationByCityName(cityName string) ([]Station, error)
	GetStationByPrefixName(cityName string) ([]Station, error)
	GetAllStations() ([]Station, error)
	UpdateStation(station *Station) error
	DeleteStation(id int) error
}

type StationDAOImpl struct {
	DB *gorm.DB
}

// NewStationDAO 创建新的 StationDAO 实例
func NewStationDAO(db *gorm.DB) StationDAO {
	return &StationDAOImpl{
		DB: db,
	}
}

var _ StationDAO = (*StationDAOImpl)(nil)

// CreateStation 创建一个新的车站记录
func (dao *StationDAOImpl) CreateStation(station *Station) error {
	return dao.DB.Create(station).Error
}

// GetStationByID 根据 ID 获取车站信息
func (dao *StationDAOImpl) GetStationByID(id int) (*Station, error) {
	var station Station
	result := dao.DB.First(&station, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &station, nil
}

// GetAllStations 获取所有车站信息
func (dao *StationDAOImpl) GetAllStations() ([]Station, error) {
	var stations []Station
	result := dao.DB.Find(&stations)
	if result.Error != nil {
		return nil, result.Error
	}
	return stations, nil
}

// UpdateStation 更新车站信息
func (dao *StationDAOImpl) UpdateStation(station *Station) error {
	result := dao.DB.Save(station)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// DeleteStation 删除车站信息
func (dao *StationDAOImpl) DeleteStation(id int) error {
	result := dao.DB.Delete(&Station{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (dao *StationDAOImpl) GetStationByName(name string) (*Station, error) {
	var station Station
	result := dao.DB.Where("station_name = ?", name).Find(&station)
	if result.Error != nil {
		return nil, result.Error
	}
	return &station, nil
}

func (dao *StationDAOImpl) GetStationByCityName(cityName string) ([]Station, error) {
	var stations []Station
	result := dao.DB.Where("city_name = ?", cityName).Find(&stations)
	if result.Error != nil {
		return nil, result.Error
	}
	return stations, nil
}

func (dao *StationDAOImpl) GetStationByPrefixName(cityName string) ([]Station, error) {
	var stations []Station
	result := dao.DB.Where("city_name Like ?%", cityName).Find(&stations)
	if result.Error != nil {
		return nil, result.Error
	}
	return stations, nil
}
