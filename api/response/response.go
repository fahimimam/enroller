package response

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ServeJSON a utility func which serves json to http client
func ServeJSON(w http.ResponseWriter, status int, message string, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := &Response{
		Status:  status,
		Data:    data,
		Message: message,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return err
	}

	return nil
}

func ServeJSONData(w http.ResponseWriter, data interface{}, status int) {
	res := responseData{
		status: status,
		Data:   data,
	}
	res.serveJSON(w)
}

type responseData struct {
	status int
	Data   interface{} `json:"data,omitempty"`
}

func (res *responseData) serveJSON(w http.ResponseWriter) {
	if res.status == 0 {
		res.status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(res.status)
	if err := json.NewEncoder(w).Encode(res); err != nil {
		panic(err)
	}
}
