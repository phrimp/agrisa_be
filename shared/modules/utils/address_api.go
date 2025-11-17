package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type Province struct {
	Code                string `json:"code"`
	Name                string `json:"name"`
	EnglishName         string `json:"englishName"`
	AdministrativeLevel string `json:"administrativeLevel"`
	Decree              string `json:"decree"`
}

type Commune struct {
	Code                string `json:"code"`
	Name                string `json:"name"`
	EnglishName         string `json:"englishName"`
	AdministrativeLevel string `json:"administrativeLevel"`
	ProvinceCode        string `json:"provinceCode"`
	ProvinceName        string `json:"provinceName"`
	Decree              string `json:"decree"`
}

// ProvinceResponse represents the API response structure
type ProvinceResponse struct {
	RequestID string     `json:"requestId"`
	Provinces []Province `json:"provinces"`
}

type CommuneResponse struct {
	RequestID string    `json:"requestId"`
	Communes  []Commune `json:"communes"`
}

func GetProvinceInfo() ([]Province, error) {
	url := "https://production.cas.so/address-kit/2025-07-01/provinces"

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make GET request
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi gọi API: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API trả về status code: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi đọc response: %w", err)
	}

	// Parse JSON response
	var provinceResponse ProvinceResponse
	if err := json.Unmarshal(body, &provinceResponse); err != nil {
		return nil, fmt.Errorf("lỗi khi parse JSON: %w", err)
	}

	return provinceResponse.Provinces, nil
}

// GetProvinceByCode is a helper function to get province details by code
func GetProvinceByCode(provinceCode string) (*Province, error) {
	provinces, err := GetProvinceInfo()
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(provinceCode)
	for _, province := range provinces {
		if province.Code == trimmed {
			return &province, nil
		}
	}

	return nil, fmt.Errorf("không tìm thấy tỉnh với mã: %s", provinceCode)
}

// getWardInfo calls the commune/ward API and returns the list of communes/wards
func GetWardInfo(provinceCode string) ([]Commune, error) {
	trimmedProvinceCode := strings.TrimSpace(provinceCode)
	url := fmt.Sprintf("https://production.cas.so/address-kit/2025-07-01/provinces/%s/communes", trimmedProvinceCode)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make GET request
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Error calling commune API: %s", err.Error())
		return nil, fmt.Errorf("lỗi khi gọi API: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("Commune API returned status code: %d", resp.StatusCode)
		return nil, fmt.Errorf("API trả về status code: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading commune API response: %s", err.Error())
		return nil, fmt.Errorf("lỗi khi đọc response: %w", err)
	}

	// Parse JSON response
	var communeResponse CommuneResponse
	if err := json.Unmarshal(body, &communeResponse); err != nil {
		log.Printf("Error parsing commune API JSON: %s", err.Error())
		return nil, fmt.Errorf("lỗi khi parse JSON: %w", err)
	}

	return communeResponse.Communes, nil
}

// GetCommuneByCode is a helper function to get commune/ward details by code
func GetCommuneByCode(provinceCode string, wardCode string) (*Commune, error) {
	communes, err := GetWardInfo(provinceCode)
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(wardCode)
	for _, commune := range communes {
		if commune.Code == trimmed {
			return &commune, nil
		}
	}

	return nil, fmt.Errorf("không tìm thấy phường/xã với mã: %s", wardCode)
}

func GetCentralMeridianByAddress(address string) float64 {
	// Bảng tra cứu kinh tuyến trục theo tỉnh/thành phố
	meridianMap := map[string]float64{
		"An Giang":          104.75,
		"Bà Rịa - Vũng Tàu": 107.75,
		"Bình Dương":        105.75,
		"Bình Phước":        106.25,
		"Bình Thuận":        108.50,
		"Bình Định":         108.25,
		"Bạc Liêu":          105.00,
		"Bắc Giang":         107.00,
		"Bắc Kạn":           106.50,
		"Bắc Ninh":          105.50,
		"Bến Tre":           105.75,
		"Cao Bằng":          105.75,
		"Cà Mau":            104.50,
		"Cần Thơ":           105.00,
		"Gia Lai":           108.50,
		"Hoà Bình":          106.00,
		"Hà Giang":          105.50,
		"Hà Nam":            105.00,
		"Hà Nội":            105.00,
		"Hà Tĩnh":           105.50,
		"Hưng Yên":          105.50,
		"Hải Dương":         105.50,
		"Hải Phòng":         105.75,
		"Hậu Giang":         105.00,
		"Hồ Chí Minh":       105.75,
		"Khánh Hoà":         108.25,
		"Kiên Giang":        104.50,
		"Kon Tum":           107.50,
		"Lai Châu":          103.00,
		"Long An":           105.75,
		"Lào Cai":           104.75,
		"Lâm Đồng":          107.75,
		"Lạng Sơn":          107.25,
		"Nam Định":          105.50,
		"Nghệ An":           104.75,
		"Ninh Bình":         105.00,
		"Ninh Thuận":        108.25,
		"Phú Thọ":           104.75,
		"Phú Yên":           108.50,
		"Quảng Bình":        106.00,
		"Quảng Nam":         107.75,
		"Quảng Ngãi":        108.00,
		"Quảng Ninh":        107.75,
		"Quảng Trị":         106.25,
		"Sóc Trăng":         105.50,
		"Sơn La":            104.00,
		"Thanh Hoá":         105.00,
		"Thái Bình":         105.50,
		"Thái Nguyên":       106.50,
		"Thừa Thiên Huế":    107.00,
		"Tiền Giang":        105.75,
		"Trà Vinh":          105.50,
		"Tuyên Quang":       106.00,
		"Tây Ninh":          105.50,
		"Vĩnh Long":         105.50,
		"Vĩnh Phúc":         105.00,
		"Yên Bái":           104.75,
		"Điện Biên":         103.00,
		"Đà Nẵng":           107.75,
		"Đắk Lắk":           108.50,
		"Đắk Nông":          108.50,
		"Đồng Nai":          107.75,
		"Đồng Tháp":         105.00,
	}

	address = strings.TrimSpace(address)

	for province, meridian := range meridianMap {
		if strings.Contains(address, province) {
			return meridian
		}
	}

	return 0
}
