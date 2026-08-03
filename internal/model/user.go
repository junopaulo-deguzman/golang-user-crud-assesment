package model

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
}

type UserInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
}

type UserPatch struct {
	Username *string `json:"username"`
	Email    *string `json:"email"`
	Age      *int    `json:"age"`
}
