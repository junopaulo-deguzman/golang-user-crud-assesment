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
	"strings"

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
	PatchUserByID(context.Context, int64, model.UserPatch) (model.User, error)
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
	if err := decodeJSONBody(r, &input); err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			err.Error(),
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
	if err := decodeJSONBody(r, &input); err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			err.Error(),
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

func (h *Handler) PatchUser(w http.ResponseWriter, r *http.Request) {
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

	var input model.UserPatch
	if err := decodeJSONBody(r, &input); err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	if err := validateUserPatch(input); err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	user, err := h.users.PatchUserByID(r.Context(), idInt, input)

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

func decodeJSONBody(r *http.Request, destination any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("Invalid request payload")
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("Request body must contain only one JSON object")
	}

	return nil
}

func validateUserInput(input model.UserInput) error {
	if err := validateUsername(input.Username); err != nil {
		return err
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

func validateUserPatch(input model.UserPatch) error {

	if input.Username == nil && input.Email == nil && input.Age == nil {
		return fmt.Errorf("at least one field (username, email, age) must be provided for patching")
	}

	if input.Username != nil {
		if err := validateUsername(*input.Username); err != nil {
			return err
		}
	}
	if input.Email != nil {
		if err := validateEmail(*input.Email); err != nil {
			return err
		}
	}
	if input.Age != nil {
		if *input.Age <= 0 {
			return fmt.Errorf("age must be a positive integer")
		}
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

	if len(value) > 254 {
		return fmt.Errorf("email must not exceed 254 characters")
	}

	return nil
}

func validateUsername(value string) error {
	trimmedValue := strings.TrimSpace(value)

	if len(trimmedValue) == 0 {
		return fmt.Errorf("username is required")
	}

	if len(trimmedValue) < 3 || len(trimmedValue) > 100 {
		return fmt.Errorf("username must be between 3 and 100 characters")
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
