package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang-user-crud-assesment/internal/model"

	"github.com/go-sql-driver/mysql"
)

type stubUserRepository struct {
	createUser func(context.Context, model.CreateUserInput) (model.User, error)
}

func (s stubUserRepository) CreateUser(
	ctx context.Context,
	input model.CreateUserInput,
) (model.User, error) {
	return s.createUser(ctx, input)
}

func TestCreateUserSuccess(t *testing.T) {
	want := model.User{
		ID:       42,
		Username: "juno",
		Email:    "juno@example.com",
		Age:      30,
	}
	repository := stubUserRepository{
		createUser: func(_ context.Context, input model.CreateUserInput) (model.User, error) {
			if input.Username != want.Username || input.Email != want.Email || input.Age != want.Age {
				t.Fatalf("CreateUser input = %+v", input)
			}
			return want, nil
		},
	}

	response := performCreateUserRequest(
		t,
		repository,
		`{"username":"juno","email":"juno@example.com","age":30}`,
	)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	var got model.User
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got != want {
		t.Errorf("response = %+v, want %+v", got, want)
	}
}

func TestCreateUserRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "malformed JSON",
			body:       `{"username":`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request payload",
		},
		{
			name:       "trailing JSON",
			body:       `{"username":"juno","email":"juno@example.com","age":30}{}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Request body must contain only one JSON object",
		},
		{
			name:       "unknown field",
			body:       `{"username":"juno","email":"juno@example.com","age":30,"admin":true}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request payload",
		},
		{
			name:       "missing username",
			body:       `{"email":"juno@example.com","age":30}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "username is required",
		},
		{
			name:       "missing email",
			body:       `{"username":"juno","age":30}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "email is required",
		},
		{
			name:       "invalid email",
			body:       `{"username":"juno","email":"not-an-email","age":30}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid email format",
		},
		{
			name:       "display-name email",
			body:       `{"username":"juno","email":"Juno <juno@example.com>","age":30}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid email format",
		},
		{
			name:       "non-positive age",
			body:       `{"username":"juno","email":"juno@example.com","age":0}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "age must be a positive integer",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := stubUserRepository{
				createUser: func(context.Context, model.CreateUserInput) (model.User, error) {
					t.Fatal("repository should not be called for an invalid request")
					return model.User{}, nil
				},
			}

			response := performCreateUserRequest(t, repository, test.body)

			assertErrorResponse(t, response, test.wantStatus, test.wantError)
		})
	}
}

func TestCreateUserRepositoryErrors(t *testing.T) {
	tests := []struct {
		name       string
		repository stubUserRepository
		wantStatus int
		wantError  string
	}{
		{
			name: "duplicate user",
			repository: stubUserRepository{
				createUser: func(context.Context, model.CreateUserInput) (model.User, error) {
					return model.User{}, &mysql.MySQLError{
						Number:  1062,
						Message: "Duplicate entry",
					}
				},
			},
			wantStatus: http.StatusConflict,
			wantError:  "Username or email already exists",
		},
		{
			name: "unexpected database error",
			repository: stubUserRepository{
				createUser: func(context.Context, model.CreateUserInput) (model.User, error) {
					return model.User{}, errors.New("database unavailable")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to create user",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performCreateUserRequest(
				t,
				test.repository,
				`{"username":"juno","email":"juno@example.com","age":30}`,
			)

			assertErrorResponse(t, response, test.wantStatus, test.wantError)
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "plain address", value: "juno@example.com"},
		{name: "malformed address", value: "not-an-email", wantErr: true},
		{name: "display name", value: "Juno <juno@example.com>", wantErr: true},
		{name: "surrounding whitespace", value: " juno@example.com ", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateEmail(test.value)
			if (err != nil) != test.wantErr {
				t.Errorf("validateEmail(%q) error = %v, wantErr %v", test.value, err, test.wantErr)
			}
		})
	}
}

func performCreateUserRequest(
	t *testing.T,
	repository UserRepository,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	response := httptest.NewRecorder()
	New(repository).CreateUser(response, request)

	return response
}

func assertErrorResponse(
	t *testing.T,
	response *httptest.ResponseRecorder,
	wantStatus int,
	wantMessage string,
) {
	t.Helper()

	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d", response.Code, wantStatus)
	}

	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body["error"] != wantMessage {
		t.Errorf("error = %q, want %q", body["error"], wantMessage)
	}
}
