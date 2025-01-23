package test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"screenshoter/internal"
	"screenshoter/internal/screenshot"
	"testing"
)

const successEndpoint = "https://google.com"
const failedEndpoint = "http://localhost:81"

type requestCase struct {
	name           string
	method         string
	authHeader     string
	body           internal.Request
	wantStatusCode int
	wantBody       internal.ErrorResponse
}

var requestCases = []requestCase{
	{
		name:           "Bad method",
		method:         http.MethodGet,
		authHeader:     "Bearer " + os.Getenv("APP_TOKEN"),
		body:           internal.Request{URL: successEndpoint},
		wantStatusCode: http.StatusMethodNotAllowed,
		wantBody: internal.ErrorResponse{
			Error: internal.OnlyPostAllowed,
		},
	},
	{
		name:           "Invalid auth header",
		method:         http.MethodPost,
		authHeader:     "Bearer 12345",
		body:           internal.Request{URL: successEndpoint},
		wantStatusCode: http.StatusUnauthorized,
		wantBody: internal.ErrorResponse{
			Error: internal.TokenInvalid,
		},
	},
}

var requestWithScreenshotCases = []requestCase{
	{
		name:           "Timeout",
		method:         http.MethodPost,
		authHeader:     "Bearer " + os.Getenv("APP_TOKEN"),
		body:           internal.Request{URL: failedEndpoint},
		wantStatusCode: http.StatusGatewayTimeout,
	},
	{
		name:           "Success",
		method:         http.MethodPost,
		authHeader:     "Bearer " + os.Getenv("APP_TOKEN"),
		body:           internal.Request{URL: successEndpoint},
		wantStatusCode: http.StatusOK,
	},
}

func TestRequestHandler(t *testing.T) {
	for _, c := range requestCases {
		t.Run(c.name, func(t *testing.T) {
			bodyJSON, err := json.Marshal(c.body)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			resp := httptest.NewRecorder()
			req := httptest.NewRequest(c.method, "/make-request", bytes.NewBuffer(bodyJSON))
			req.Header.Set("Authorization", c.authHeader)

			internal.Handle(resp, req)

			if resp.Code != c.wantStatusCode {
				t.Errorf("Want status code %d, got %d", c.wantStatusCode, resp.Code)
			}

			var gotResponse internal.ErrorResponse
			if err := json.NewDecoder(resp.Body).Decode(&gotResponse); err != nil {
				log.Fatalf("Failed to decode response: %v", err)
			}

			if gotResponse != c.wantBody {
				t.Errorf("Want response %+v, got %+v", c.wantBody, gotResponse)
			}
		})
	}
}

func TestCaptureScreenshot(t *testing.T) {
	mockCapture := new(screenshot.MockCapture)
	mockCapture.On("Capture", mock.Anything, successEndpoint).Return([]byte("fake_screenshot"), nil)
	mockCapture.On("Capture", mock.Anything, failedEndpoint).Return([]byte{}, context.DeadlineExceeded)

	internal.SetCapture(mockCapture)

	t.Cleanup(func() {
		internal.SetCapture(nil)
	})

	for _, c := range requestWithScreenshotCases {
		t.Run(c.name, func(t *testing.T) {
			bodyJSON, err := json.Marshal(c.body)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			resp := httptest.NewRecorder()
			req := httptest.NewRequest(c.method, "/make-request", bytes.NewBuffer(bodyJSON))
			req.Header.Set("Authorization", c.authHeader)

			internal.Handle(resp, req)

			assert.Equal(t, c.wantStatusCode, resp.Code, "expected status code")
		})
	}

	mockCapture.AssertExpectations(t)
}
