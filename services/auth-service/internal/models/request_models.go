package models

type LoginRequest struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email      string `json:"email" binding:"required"`
	Phone      string `json:"phone" binding:"required"`
	Password   string `json:"password" binding:"required"`
	NationalID string `json:"national_id" binding:"required"`
}

type LoginResponse struct {
	User        *User        `json:"user"`
	Session     *UserSession `json:"session"`
	AccessToken string       `json:"access_token"`
}
