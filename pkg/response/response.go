package response

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, statusCode int, success bool, message string, data interface{}, apiErr *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := APIResponse{
		Success: success,
		Message: message,
		Data:    data,
		Error:   apiErr,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func OK(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusOK, true, message, data, nil)
}

func Created(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusCreated, true, message, data, nil)
}

func BadRequest(w http.ResponseWriter, message string, errCode string, details string) {
	JSON(w, http.StatusBadRequest, false, message, nil, &APIError{Code: errCode, Details: details})
}

func Unauthorized(w http.ResponseWriter, message string, details string) {
	JSON(w, http.StatusUnauthorized, false, message, nil, &APIError{Code: "UNAUTHORIZED", Details: details})
}

func Conflict(w http.ResponseWriter, message string, errCode string, details string) {
	JSON(w, http.StatusConflict, false, message, nil, &APIError{Code: errCode, Details: details})
}

func InternalServerError(w http.ResponseWriter, message string) {
	JSON(w, http.StatusInternalServerError, false, message, nil, &APIError{Code: "INTERNAL_SERVER_ERROR", Details: "an unexpected error occurred"})
}
