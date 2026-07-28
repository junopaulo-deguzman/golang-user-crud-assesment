package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicAuth(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		password      string
		setCredential bool
		wantStatus    int
		wantCalled    bool
	}{
		{
			name:          "valid credentials",
			username:      "api-user",
			password:      "secret",
			setCredential: true,
			wantStatus:    http.StatusOK,
			wantCalled:    true,
		},
		{
			name:       "missing credentials",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:          "incorrect username",
			username:      "someone-else",
			password:      "secret",
			setCredential: true,
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "incorrect password",
			username:      "api-user",
			password:      "wrong",
			setCredential: true,
			wantStatus:    http.StatusUnauthorized,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})
			handler := BasicAuth("api-user", "secret", next)
			request := httptest.NewRequest(http.MethodPost, "/users", nil)
			if test.setCredential {
				request.SetBasicAuth(test.username, test.password)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if called != test.wantCalled {
				t.Errorf("next handler called = %v, want %v", called, test.wantCalled)
			}
			if test.wantStatus == http.StatusUnauthorized {
				if got := response.Header().Get("WWW-Authenticate"); got == "" {
					t.Error("WWW-Authenticate header is missing")
				}
			}
		})
	}
}
