package repository

import (
	"context"
	"database/sql"
	"golang-user-crud-assesment/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(
	ctx context.Context,
	input model.CreateUserInput,
) (model.User, error) {
	result, err := r.db.ExecContext(
		ctx,
		"INSERT INTO users (username, email, age) VALUES (?, ?, ?)",
		input.Username,
		input.Email,
		input.Age,
	)
	if err != nil {
		return model.User{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return model.User{}, err
	}

	return model.User{
		ID:       int64(id),
		Username: input.Username,
		Email:    input.Email,
		Age:      input.Age,
	}, nil
}
