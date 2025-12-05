package services

import (
	"profile-service/internal/models"
	"profile-service/internal/repository"
	"utils"
)

type UserService struct {
	repo repository.IUserRepository
}

type IUserService interface {
	GetUserProfileByUserID(userID string) (*models.UserProfile, error)
	CreateUserProfile(req *models.CreateUserProfileRequest, createdByID, createdByName string) error
	UpdateUserProfile(updateProfileRequestBody map[string]interface{}, userID, updatedByName string) (*models.UserProfile, error)
	GetUserProfilesByPartnerID(partnerID string) ([]models.UserProfile, error)
	GetUserBankInfoByUserIDs(userIDs []string) ([]models.UserBankInfo, error)
}

func NewUserService(repo repository.IUserRepository) IUserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) GetUserProfileByUserID(userID string) (*models.UserProfile, error) {
	return s.repo.GetUserProfileByUserID(userID)
}

func (s *UserService) CreateUserProfile(req *models.CreateUserProfileRequest, createdByID, createdByName string) error {
	return s.repo.CreateUserProfile(req, createdByID, createdByName)
}

var allowedUpdateUserProfileFields = map[string]bool{
	"full_name":         true,
	"display_name":      true,
	"date_of_birth":     true,
	"gender":            true,
	"nationality":       true,
	"primary_phone":     true,
	"alternate_phone":   true,
	"permanent_address": true,
	"current_address":   true,
	"province_code":     true,
	"province_name":     true,
	"district_code":     true,
	"district_name":     true,
	"ward_code":         true,
	"ward_name":         true,
	"postal_code":       true,
}

var arrayUserProfileFields = map[string]bool{}

func (s *UserService) UpdateUserProfile(updateProfileRequestBody map[string]interface{}, userID, updatedByName string) (*models.UserProfile, error) {
	// check if insurance partner profile exists
	_, err := s.repo.GetUserProfileByUserID(userID)
	if err != nil {
		return nil, err
	}

	specialFields := make(map[string]*utils.FieldTransformer)

	queryBuilderResult, err := utils.BuildDynamicUpdateQuery("user_profiles", updateProfileRequestBody, allowedUpdateUserProfileFields, arrayUserProfileFields, specialFields, "user_id", userID, true, userID, "last_updated_by")
	if err != nil {
		return nil, err
	}

	err = s.repo.UpdateUserProfile(queryBuilderResult.Query, queryBuilderResult.Args...)
	if err != nil {
		return nil, err
	}
	updatedProfile, err := s.repo.GetUserProfileByUserID(userID)
	if err != nil {
		return nil, err
	}

	return updatedProfile, nil
}

func (s *UserService) GetUserProfilesByPartnerID(partnerID string) ([]models.UserProfile, error) {
	return s.repo.GetUserProfilesByPartnerID(partnerID)
}
func (s *UserService) GetUserBankInfoByUserIDs(userIDs []string) ([]models.UserBankInfo, error) {
	return s.repo.GetUserBankInfoByUserIDs(userIDs)
}
