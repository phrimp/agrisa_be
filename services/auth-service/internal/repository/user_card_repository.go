package repository

import (
	"auth-service/internal/models"

	"github.com/jmoiron/sqlx"
)

type IUserCardRepository interface {
	CreateUserCard(userCard *models.UserCard) (*models.UserCard, error)
	GetUserCardByUserID(userID string) (*models.UserCard, error)
}

type UserCardRepository struct {
	db *sqlx.DB
}

func NewUserCardRepository(db *sqlx.DB) IUserCardRepository {
	return &UserCardRepository{
		db: db,
	}
}
func (u *UserCardRepository) CreateUserCard(userCard *models.UserCard) (*models.UserCard, error) {
	_, err := u.db.NamedExec(`INSERT INTO user_card (national_id, name, dob, sex, nationality, home, address, doe, number_of_name_lines, features, issue_date, mrz, issue_loc, image_front, image_back, user_id)
		VALUES (:national_id, :name, :dob, :sex, :nationality, :home, :address, :doe, :number_of_name_lines, :features, :issue_date, :mrz, :issue_loc, :image_front, :image_back, :user_id)`, userCard)
	if err != nil {
		return nil, err
	}
	return userCard, nil
}

func (u *UserCardRepository) GetUserCardByUserID(userID string) (*models.UserCard, error) {
	var userCard models.UserCard
	err := u.db.Get(&userCard, "SELECT * FROM user_card WHERE user_id=$1", userID)
	if err != nil {
		return nil, err
	}
	return &userCard, nil
}
