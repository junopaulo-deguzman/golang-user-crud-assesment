package repository

import (
	"context"
	"database/sql"
	"errors"
	"golang-user-crud-assesment/internal/model"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(
	ctx context.Context,
	input model.UserInput,
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

func (r *UserRepository) GetUserByID(
	ctx context.Context,
	id int64,
) (model.User, error) {
	var user model.User
	err := r.db.QueryRowContext(
		ctx,
		"SELECT id, username, email, age FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Age)

	if err != nil {
		if err == sql.ErrNoRows {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, err
	}

	return user, nil
}

func (r *UserRepository) UpdateUserByID(
	ctx context.Context,
	id int64,
	input model.UserInput,
) (model.User, error) {
	_, err := r.db.ExecContext(
		ctx,
		"UPDATE users SET username = ?, email = ?, age = ? WHERE id = ?",
		input.Username,
		input.Email,
		input.Age,
		id,
	)
	if err != nil {
		return model.User{}, err
	}

	return r.GetUserByID(ctx, id)
}

func (r *UserRepository) DeleteUserByID(
	ctx context.Context,
	id int64,
) error {
	result, err := r.db.ExecContext(
		ctx,
		"DELETE FROM users WHERE id = ?",
		id,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}
