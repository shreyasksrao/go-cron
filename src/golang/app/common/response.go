package common

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error"`
}

func WriteOkResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Status: "OK",
		Data:   data,
		Error:  "",
	}
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "\t")
	if err := encoder.Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func WriteErrorResponse(w http.ResponseWriter, errMsg string, status string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Status: status,
		Error:  errMsg,
	}
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "\t")
	if err := encoder.Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
