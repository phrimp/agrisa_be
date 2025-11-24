package repository

import (
	"fmt"
	"log"
	"profile-service/internal/models"
	"utils"

	"github.com/jmoiron/sqlx"
)

type IUserRepository interface {
	GetUserProfileByUserID(userID string) (*models.UserProfile, error)
	CreateUserProfile(req *models.CreateUserProfileRequest, createdByID, createdByName string) error
	UpdateUserProfile(query string, args ...interface{}) error
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

func (r *UserRepository) CreateUserProfile(req *models.CreateUserProfileRequest, createdByID, createdByName string) error {
	log.Printf("CreateUserProfile called with createdByID: %s, createdByName: %s", createdByID, createdByName)

	query := `
        INSERT INTO user_profiles (
            user_id,
            role_id,
            partner_id,
            full_name,
            display_name,
            date_of_birth,
            gender,
            nationality,
            email,
            primary_phone,
            alternate_phone,
            permanent_address,
            current_address,
            province_code,
            province_name,
            district_code,
            district_name,
            ward_code,
            ward_name,
            postal_code,
            last_updated_by,
            last_updated_by_name
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
            $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
            $21, $22
        )
    `

	err := utils.ExecWithCheck(
		r.db,
		query,
		utils.ExecInsert,
		req.UserID,
		req.RoleID,
		req.PartnerID,
		req.FullName,
		req.DisplayName,
		req.DateOfBirth,
		req.Gender,
		req.Nationality,
		req.Email,
		req.PrimaryPhone,
		req.AlternatePhone,
		req.PermanentAddress,
		req.CurrentAddress,
		req.ProvinceCode,
		req.ProvinceName,
		req.DistrictCode,
		req.DistrictName,
		req.WardCode,
		req.WardName,
		req.PostalCode,
		createdByID,
		createdByName,
	)

	if err != nil {
		log.Printf("Error creating user profile: %s", err.Error())
		return fmt.Errorf("failed to create user profile: %w", err)
	}

	return nil
}

func (r *UserRepository) UpdateUserProfile(query string, args ...interface{}) error {
	if err := utils.ExecWithCheck(r.db, query, utils.ExecUpdate, args...); err != nil {
		return fmt.Errorf("failed to update insurance partner: %w", err)
	}
	return nil
}
