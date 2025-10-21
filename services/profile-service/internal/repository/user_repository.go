package repository

import (
	"log"
	"profile-service/internal/models"

	"github.com/jmoiron/sqlx"
)

type IUserRepository interface {
	GetUserProfileByUserID(userID string) (*models.UserProfile, error)
}

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) IUserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) GetUserProfileByUserID(userID string) (*models.UserProfile, error) {
	var userProfile models.UserProfile
	err := r.db.Get(&userProfile, "SELECT * FROM user_profiles WHERE user_id = $1", userID)
	if err != nil {
		log.Printf("Error fetching user profile by userID %s: %v", userID, err)
		return nil, err
	}
	return &userProfile, nil
}
