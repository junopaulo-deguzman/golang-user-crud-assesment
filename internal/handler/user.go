package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang-user-crud-assesment/internal/repository"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strconv"

	"golang-user-crud-assesment/internal/model"
	"golang-user-crud-assesment/internal/response"

	"github.com/go-sql-driver/mysql"
)

type Handler struct {
	users UserRepository
}

type UserRepository interface {
	CreateUser(context.Context, model.UserInput) (model.User, error)
	GetUserByID(context.Context, int64) (model.User, error)
	UpdateUserByID(context.Context, int64, model.UserInput) (model.User, error)
	DeleteUserByID(context.Context, int64) error
}

func New(users UserRepository) *Handler {
	return &Handler{users: users}
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	var id string
	if id = r.PathValue("id"); id == "" {
		response.Error(
			w,
			http.StatusBadRequest,
			"User ID is required",
		)
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil || idInt <= 0 {
		response.Error(
			w,
			http.StatusBadRequest,
			"Invalid user ID",
		)
		return
	}

	user, err := h.users.GetUserByID(r.Context(), idInt)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			response.Error(
				w,
				http.StatusNotFound,
				"User not found",
			)
			return
		}
		response.Error(
			w,
			http.StatusInternalServerError,
			"Failed to get user",
		)
		return
	}

	response.JSON(
		w,
		http.StatusOK,
		user,
	)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var input model.UserInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"Invalid request payload",
		)
		return
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		response.Error(
			w,
			http.StatusBadRequest,
			"Request body must contain only one JSON object",
		)
		return
	}

	if err := validateUserInput(input); err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	user, err := h.users.CreateUser(r.Context(), input)
	if err != nil {
		status, message, handled := checkSqlDuplicateError(err)
		if handled {
			response.Error(
				w,
				status,
				message,
			)
			return
		}
		response.Error(
			w,
			http.StatusInternalServerError,
			"Failed to create user",
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(user); err != nil {
		log.Printf("failed to write response: %v", err)
	}

	log.Printf("Received request to create user: %+v", input)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var id string
	if id = r.PathValue("id"); id == "" {
		response.Error(
			w,
			http.StatusBadRequest,
			"User ID is required",
		)
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil || idInt <= 0 {
		response.Error(
			w,
			http.StatusBadRequest,
			"Invalid user ID",
		)
		return
	}

	var input model.UserInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"Invalid request payload",
		)
		return
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		response.Error(
			w,
			http.StatusBadRequest,
			"Request body must contain only one JSON object",
		)
		return
	}

	if err := validateUserInput(input); err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	user, err := h.users.UpdateUserByID(r.Context(), idInt, input)

	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			response.Error(
				w,
				http.StatusNotFound,
				"User not found",
			)
			return
		}

		if status, message, handled := checkSqlDuplicateError(err); handled {
			response.Error(
				w,
				status,
				message,
			)
			return
		}

		response.Error(
			w,
			http.StatusInternalServerError,
			"Failed to update user",
		)
		return
	}

	response.JSON(
		w,
		http.StatusOK,
		user,
	)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var id string
	if id = r.PathValue("id"); id == "" {
		response.Error(
			w,
			http.StatusBadRequest,
			"User ID is required",
		)
		return
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil || idInt <= 0 {
		response.Error(
			w,
			http.StatusBadRequest,
			"Invalid user ID",
		)
		return
	}

	err = h.users.DeleteUserByID(r.Context(), idInt)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			response.Error(
				w,
				http.StatusNotFound,
				"User not found",
			)
			return
		}
		response.Error(
			w,
			http.StatusInternalServerError,
			"Failed to delete user",
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validateUserInput(input model.UserInput) error {
	if input.Username == "" {
		return fmt.Errorf("username is required")
	}
	if input.Email == "" {
		return fmt.Errorf("email is required")
	}
	if err := validateEmail(input.Email); err != nil {
		return fmt.Errorf("%v", err)
	}
	if input.Age <= 0 {
		return fmt.Errorf("age must be a positive integer")
	}
	return nil
}

func validateEmail(value string) error {
	address, err := mail.ParseAddress(value)
	if err != nil {
		return fmt.Errorf("invalid email format")
	}

	if address.Address != value {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

func checkSqlDuplicateError(err error) (status int, message string, handled bool) {
	var mysqlErr *mysql.MySQLError

	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return http.StatusConflict, "Username or email already exists", true
	}

	return 0, "", false
}
