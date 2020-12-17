package routes

import (
	"encoding/json"
	"net/http"
)

func WriteResponse(w http.ResponseWriter, h func() (interface{}, int, error)) {
	response, statusCode, err := h()
	if err != nil {
		response = struct {
			Error string `json:"error"`
		}{Error: err.Error()}
	}

	data, err := json.Marshal(response)
	if err != nil {
		data, _ = json.Marshal(struct {
			Error string `json:"error"`
		}{Error: err.Error()})
		statusCode = http.StatusInternalServerError
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if _, err := w.Write(data); err != nil {
		return
	}
}
