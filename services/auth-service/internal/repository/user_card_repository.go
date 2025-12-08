package repository

import (
	"auth-service/internal/models"
	"fmt"
	"log"
	"strings"

	"github.com/jmoiron/sqlx"
)

type IUserCardRepository interface {
	CreateUserCard(userCard *models.UserCard) (*models.UserCard, error)
	GetUserCardByUserID(userID string) (*models.UserCard, error)
	UpdateUserCardByUserID(userID string, req models.UpdateUserCardRequest) error
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

func (u *UserCardRepository) UpdateUserCardByUserID(userID string, req models.UpdateUserCardRequest) error {
	updates := make(map[string]interface{})

	if req.NationalID != nil {
		updates["national_id"] = *req.NationalID
	}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.DOB != nil {
		updates["dob"] = *req.DOB
	}
	if req.Sex != nil {
		updates["sex"] = *req.Sex
	}
	if req.Nationality != nil {
		updates["nationality"] = *req.Nationality
	}
	if req.Home != nil {
		updates["home"] = *req.Home
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.DOE != nil {
		updates["doe"] = *req.DOE
	}
	if req.NumberOfNameLines != nil {
		updates["number_of_name_lines"] = *req.NumberOfNameLines
	}
	if req.Features != nil {
		updates["features"] = *req.Features
	}
	if req.IssueDate != nil {
		updates["issue_date"] = *req.IssueDate
	}
	if req.MRZ != nil {
		updates["mrz"] = *req.MRZ
	}
	if req.IssueLoc != nil {
		updates["issue_loc"] = *req.IssueLoc
	}
	if req.ImageFront != nil {
		updates["image_front"] = *req.ImageFront
	}
	if req.ImageBack != nil {
		updates["image_back"] = *req.ImageBack
	}

	if len(updates) == 0 {
		log.Printf("no fields to update")
		return fmt.Errorf("no fields to update")
	}

	setClauses := make([]string, 0, len(updates))
	args := make(map[string]interface{})

	for column, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = :%s", column, column))
		args[column] = value
	}

	args["user_id"] = userID

	query := fmt.Sprintf(
		"UPDATE user_card SET %s WHERE user_id = :user_id",
		strings.Join(setClauses, ", "),
	)

	result, err := u.db.NamedExec(query, args)
	if err != nil {
		log.Printf("failed to update user card: %v", err)
		return fmt.Errorf("failed to update user card: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("no user card found with user_id: %s", userID)
		return fmt.Errorf("not_found:no user card found with user_id: %s", userID)
	}

	return nil
}
