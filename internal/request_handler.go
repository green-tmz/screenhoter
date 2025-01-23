package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"screenshoter/internal/screenshot"
	"strings"
	"time"
)

type Request struct {
	URL string `json:"url"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func sendErrorResponse(w http.ResponseWriter, status int, message string) {
	response := ErrorResponse{
		Error: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

var capture screenshot.ICapture = &screenshot.Capture{}

func SetCapture(c screenshot.ICapture) {
	capture = c
}

func Handle(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		sendErrorResponse(writer, http.StatusMethodNotAllowed, OnlyPostAllowed)
		return
	}

	authHeader := request.Header.Get("Authorization")

	if strings.TrimPrefix(authHeader, "Bearer ") != os.Getenv("APP_TOKEN") {
		sendErrorResponse(writer, http.StatusUnauthorized, TokenInvalid)
		return
	}

	var screenshotRequest Request
	err := json.NewDecoder(request.Body).Decode(&screenshotRequest)
	if err != nil {
		http.Error(writer, InvalidRequestBody, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 60*time.Second)
	defer cancel()

	recsurce, err := capture.Capture(ctx, screenshotRequest.URL)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			sendErrorResponse(writer, http.StatusGatewayTimeout, RequestTimeOut)
		default:
			sendErrorResponse(writer, http.StatusInternalServerError, InternalError)
		}

		fmt.Printf("Error capturing: %v\n", err)
		return
	}

	writer.Header().Set("Content-Type", "image/png")
	writer.Header().Set("Content-Disposition", "attachment; filename='screenshot.png'")
	writer.WriteHeader(http.StatusOK)

	_, err = writer.Write(recsurce)
	if err != nil {
		return
	}
}
