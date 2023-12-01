package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthEndpoint(t *testing.T) {
	t.Skip("Skipping integration test - requires full handler setup")
}

func TestUserResponseFormat(t *testing.T) {
	t.Run("success response structure", func(t *testing.T) {
		resp := UserResponse{
			Success: true,
			Data:    nil,
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal response: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if _, ok := result["success"]; !ok {
			t.Fatal("expected 'success' field in response")
		}
		if _, ok := result["data"]; !ok {
			t.Fatal("expected 'data' field in response")
		}
	})

	t.Run("error response structure", func(t *testing.T) {
		resp := ErrorResponse{
			Success: false,
			Error:   "Test error",
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal response: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if result["success"].(bool) != false {
			t.Fatal("expected success to be false")
		}
		if result["error"].(string) != "Test error" {
			t.Fatal("expected error message to match")
		}
	})

	t.Run("pagination meta structure", func(t *testing.T) {
		resp := ListUsersResponse{
			Success: true,
			Data:    nil,
			Meta: PaginationMeta{
				Total:  100,
				Limit:  20,
				Offset: 40,
			},
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal response: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		meta, ok := result["meta"].(map[string]interface{})
		if !ok {
			t.Fatal("expected 'meta' field in response")
		}

		if int(meta["total"].(float64)) != 100 {
			t.Fatal("expected total to be 100")
		}
		if int(meta["limit"].(float64)) != 20 {
			t.Fatal("expected limit to be 20")
		}
		if int(meta["offset"].(float64)) != 40 {
			t.Fatal("expected offset to be 40")
		}
	})
}

// TestCreateUserV1Request tests the legacy request format
// TODO(TEAM-API): Remove after v1 API deprecation
func TestCreateUserV1Request(t *testing.T) {
	req := CreateUserV1Request{
		Email:    "test@example.com",
		Name:     "John Doe",
		Password: "password123",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	var result CreateUserV1Request
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if result.Email != req.Email {
		t.Fatal("email mismatch")
	}
	if result.Name != req.Name {
		t.Fatal("name mismatch")
	}
}

func TestParseIntQuery(t *testing.T) {
	tests := []struct {
		name         string
		queryValue   string
		defaultValue int
		expected     int
	}{
		{"valid number", "25", 10, 25},
		{"empty string", "", 10, 10},
		{"invalid string", "abc", 10, 10},
		{"zero", "0", 10, 0},
		{"negative", "-5", 10, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test context with query parameter
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			req, _ := http.NewRequest("GET", "/?limit="+tt.queryValue, nil)
			c.Request = req

			h := &Handlers{}
			result := h.parseIntQuery(c, "limit", tt.defaultValue)

			if result != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestLoginRequest(t *testing.T) {
	reqBody := `{"email":"test@example.com","password":"password123"}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	var loginReq LoginV1Request
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		t.Fatalf("failed to bind request: %v", err)
	}

	if loginReq.Email != "test@example.com" {
		t.Fatal("email mismatch")
	}
	if loginReq.Password != "password123" {
		t.Fatal("password mismatch")
	}
}
