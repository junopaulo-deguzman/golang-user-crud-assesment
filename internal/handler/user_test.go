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
	"golang-user-crud-assesment/internal/repository"

	"github.com/go-sql-driver/mysql"
)

type stubUserRepository struct {
	createUser     func(context.Context, model.UserInput) (model.User, error)
	getUserByID    func(context.Context, int64) (model.User, error)
	updateUserByID func(context.Context, int64, model.UserInput) (model.User, error)
	deleteUserByID func(context.Context, int64) error
}

func (s stubUserRepository) CreateUser(
	ctx context.Context,
	input model.UserInput,
) (model.User, error) {
	return s.createUser(ctx, input)
}

func (s stubUserRepository) GetUserByID(
	ctx context.Context,
	id int64,
) (model.User, error) {
	return s.getUserByID(ctx, id)
}

func (s stubUserRepository) UpdateUserByID(
	ctx context.Context,
	id int64,
	input model.UserInput,
) (model.User, error) {
	return s.updateUserByID(ctx, id, input)
}

func (s stubUserRepository) DeleteUserByID(
	ctx context.Context,
	id int64,
) error {
	return s.deleteUserByID(ctx, id)
}

func TestCreateUserSuccess(t *testing.T) {
	want := model.User{
		ID:       42,
		Username: "juno",
		Email:    "juno@example.com",
		Age:      30,
	}
	repository := stubUserRepository{
		createUser: func(_ context.Context, input model.UserInput) (model.User, error) {
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
				createUser: func(context.Context, model.UserInput) (model.User, error) {
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
				createUser: func(context.Context, model.UserInput) (model.User, error) {
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
				createUser: func(context.Context, model.UserInput) (model.User, error) {
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

func TestGetUserByIDSuccess(t *testing.T) {
	repository := stubUserRepository{
		getUserByID: func(ctx context.Context, id int64) (model.User, error) {
			return model.User{
				ID:       id,
				Username: "juno",
				Email:    "juno@example.com",
				Age:      30,
			}, nil
		},
	}

	response := performGetUserRequest(t, repository, "1")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	var user model.User
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.ID != 1 {
		t.Errorf("user.ID = %d, want %d", user.ID, 1)
	}
	if user.Username != "juno" {
		t.Errorf("user.Username = %q, want %q", user.Username, "juno")
	}
	if user.Email != "juno@example.com" {
		t.Errorf("user.Email = %q, want %q", user.Email, "juno@example.com")
	}
	if user.Age != 30 {
		t.Errorf("user.Age = %d, want %d", user.Age, 30)
	}
}

func TestGetUserByIDNotFound(t *testing.T) {
	repository := stubUserRepository{
		getUserByID: func(ctx context.Context, id int64) (model.User, error) {
			return model.User{}, repository.ErrUserNotFound
		},
	}

	response := performGetUserRequest(t, repository, "1")

	assertErrorResponse(t, response, http.StatusNotFound, "User not found")
}

func TestGetUserByIDInvalidID(t *testing.T) {
	repository := stubUserRepository{
		getUserByID: func(ctx context.Context, id int64) (model.User, error) {
			t.Fatal("repository should not be called for an invalid ID")
			return model.User{}, nil
		},
	}

	tests := []struct {
		name string
		id   string
	}{
		{name: "non-numeric ID", id: "abc"},
		{name: "negative ID", id: "-1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performGetUserRequest(t, repository, test.id)

			assertErrorResponse(t, response, http.StatusBadRequest, "Invalid user ID")
		})
	}
}

func TestUpdateUserByIDSuccess(t *testing.T) {
	repository := stubUserRepository{
		updateUserByID: func(ctx context.Context, id int64, input model.UserInput) (model.User, error) {
			return model.User{
				ID:       id,
				Username: input.Username,
				Email:    input.Email,
				Age:      input.Age,
			}, nil
		},
	}

	response := performUpdateUserRequest(
		t,
		repository,
		"1",
		`{"username":"juno","email":"juno@example.com","age":30}`,
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	var user model.User
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.ID != 1 {
		t.Errorf("user.ID = %d, want %d", user.ID, 1)
	}
	if user.Username != "juno" {
		t.Errorf("user.Username = %q, want %q", user.Username, "juno")
	}
	if user.Email != "juno@example.com" {
		t.Errorf("user.Email = %q, want %q", user.Email, "juno@example.com")
	}
	if user.Age != 30 {
		t.Errorf("user.Age = %d, want %d", user.Age, 30)
	}
}

func TestUpdateUserByIDNotFound(t *testing.T) {
	repository := stubUserRepository{
		updateUserByID: func(ctx context.Context, id int64, input model.UserInput) (model.User, error) {
			return model.User{}, repository.ErrUserNotFound
		},
	}

	response := performUpdateUserRequest(
		t,
		repository,
		"1",
		`{"username":"juno","email":"juno@example.com","age":30}`,
	)

	assertErrorResponse(t, response, http.StatusNotFound, "User not found")
}

func TestUpdateUserByIDInvalidRequest(t *testing.T) {
	repository := stubUserRepository{}

	tests := []struct {
		name       string
		id         string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "malformed JSON",
			id:         "1",
			body:       `{"username":`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request payload",
		},
		{
			name:       "trailing JSON",
			id:         "1",
			body:       `{"username":"juno","email":"juno@example.com", "age":30}{}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Request body must contain only one JSON object",
		},
		{
			name:       "unknown field",
			id:         "1",
			body:       `{"username":"juno","email":"juno@example.com","unknown_field":"value"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request payload",
		},
		{
			name:       "missing username",
			id:         "1",
			body:       `{"email":"juno@example.com","age":30}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "username is required",
		},
		{
			name:       "missing email",
			id:         "1",
			body:       `{"username":"juno","age":30}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "email is required",
		},
		{
			name:       "invalid age",
			id:         "1",
			body:       `{"username":"juno","email":"juno@example.com","age":-1}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "age must be a positive integer",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performUpdateUserRequest(
				t,
				repository,
				test.id,
				test.body,
			)

			assertErrorResponse(t, response, test.wantStatus, test.wantError)
		})
	}
}

func TestUpdateUserByDuplicateRepositoryError(t *testing.T) {
	repository := stubUserRepository{
		updateUserByID: func(ctx context.Context, id int64, input model.UserInput) (model.User, error) {
			return model.User{}, &mysql.MySQLError{
				Number:  1062,
				Message: "Duplicate entry",
			}
		},
	}

	response := performUpdateUserRequest(
		t,
		repository,
		"1",
		`{"username":"juno","email":"juno@example.com","age":30}`,
	)

	assertErrorResponse(t, response, http.StatusConflict, "Username or email already exists")
}

func TestUpdateUserByUnexpectedRepositoryError(t *testing.T) {
	repository := stubUserRepository{
		updateUserByID: func(ctx context.Context, id int64, input model.UserInput) (model.User, error) {
			return model.User{}, errors.New("database unavailable")
		},
	}

	response := performUpdateUserRequest(
		t,
		repository,
		"1",
		`{"username":"juno","email":"juno@example.com","age":30}`,
	)

	assertErrorResponse(t, response, http.StatusInternalServerError, "Failed to update user")
}

func TestUpdateUserByIDInvalidID(t *testing.T) {
	repository := stubUserRepository{
		updateUserByID: func(ctx context.Context, id int64, input model.UserInput) (model.User, error) {
			t.Fatal("repository should not be called for an invalid ID")
			return model.User{}, nil
		},
	}

	tests := []struct {
		name string
		id   string
	}{
		{name: "non-numeric ID", id: "abc"},
		{name: "negative ID", id: "-1"},
		{name: "zero ID", id: "0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performUpdateUserRequest(
				t,
				repository,
				test.id,
				`{"username":"juno","email":"juno@example.com","age":30}`,
			)

			assertErrorResponse(t, response, http.StatusBadRequest, "Invalid user ID")
		})
	}
}

func TestDeleteUserByIDSuccess(t *testing.T) {
	repository := stubUserRepository{
		deleteUserByID: func(ctx context.Context, id int64) error {
			return nil
		},
	}

	response := performDeleteUserRequest(t, repository, "1")

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}

	if response.Body.Len() != 0 {
		t.Errorf("response body = %q, want empty", response.Body.String())
	}
}

func TestDeleteUserByIDNotFound(t *testing.T) {
	repository := stubUserRepository{
		deleteUserByID: func(ctx context.Context, id int64) error {
			return repository.ErrUserNotFound
		},
	}

	response := performDeleteUserRequest(t, repository, "1")

	assertErrorResponse(t, response, http.StatusNotFound, "User not found")
}

func TestDeleteUserByIDInvalidID(t *testing.T) {
	repository := stubUserRepository{
		deleteUserByID: func(ctx context.Context, id int64) error {
			t.Fatal("repository should not be called for an invalid ID")
			return nil
		},
	}

	tests := []struct {
		name string
		id   string
	}{
		{name: "non-numeric ID", id: "abc"},
		{name: "negative ID", id: "-1"},
		{name: "zero ID", id: "0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performDeleteUserRequest(t, repository, test.id)

			assertErrorResponse(t, response, http.StatusBadRequest, "Invalid user ID")
		})
	}
}

func TestDeleteUserByUnexpectedRepositoryError(t *testing.T) {
	repository := stubUserRepository{
		deleteUserByID: func(ctx context.Context, id int64) error {
			return errors.New("database unavailable")
		},
	}

	response := performDeleteUserRequest(t, repository, "1")

	assertErrorResponse(t, response, http.StatusInternalServerError, "Failed to delete user")
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

func performGetUserRequest(
	t *testing.T,
	repository UserRepository,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/users/"+id, nil)
	request.SetPathValue("id", id)
	response := httptest.NewRecorder()
	New(repository).GetUser(response, request)

	return response
}

func performUpdateUserRequest(
	t *testing.T,
	repository UserRepository,
	id string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPut, "/users/"+id, strings.NewReader(body))
	request.SetPathValue("id", id)
	response := httptest.NewRecorder()
	New(repository).UpdateUser(response, request)

	return response
}

func performDeleteUserRequest(
	t *testing.T,
	repository UserRepository,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodDelete, "/users/"+id, nil)
	request.SetPathValue("id", id)
	response := httptest.NewRecorder()
	New(repository).DeleteUser(response, request)

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
