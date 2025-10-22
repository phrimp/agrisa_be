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
