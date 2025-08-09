package utils

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	SuccessMessage = "Successful"
)

func GetTracingID(ctx context.Context) string {
	return middleware.GetReqID(ctx)
}

// ToMap Helper function to convert struct to map
func ToMap(data interface{}) (map[string]interface{}, error) {
	out := make(map[string]interface{})

	// Handle direct maps
	if m, ok := data.(map[string]interface{}); ok {
		return m, nil
	}

	// Handle structs using JSON marshaling
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bytes, &out); err != nil {
		return nil, err
	}

	return out, nil
}
