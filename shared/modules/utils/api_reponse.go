package utils

import "time"

type SuccessResponse struct {
	Success bool  `json:"success"`
	Data    any   `json:"data"`
	Meta    *Meta `json:"meta,omitempty"`
}

type ErrorResponse struct {
	Success bool     `json:"success"`
	Error   APIError `json:"error"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Meta struct {
	Timestamp time.Time `json:"timestamp"`
}

func CreateErrorResponse(code, message string) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error: APIError{
			Code:    code,
			Message: message,
		},
	}
}

func CreateSuccessResponse(data any) SuccessResponse {
	return SuccessResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Timestamp: time.Now(),
		},
	}
}
