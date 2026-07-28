package response

import (
	"encoding/json"
	"log"
	"net/http"
)

func JSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to write json response: %v", err)
	}
}

func Error(
	w http.ResponseWriter,
	status int,
	message string,
) {
	JSON(w, status, map[string]string{
		"error": message,
	})
}
