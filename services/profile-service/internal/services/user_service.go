package services

import (
	"profile-service/internal/models"
	"profile-service/internal/repository"
)

type UserService struct {
	repo repository.IUserRepository
}

type IUserService interface {
	GetUserProfileByUserID(userID string) (*models.UserProfile, error)
	CreateUserProfile(req *models.CreateUserProfileRequest, createdByID, createdByName string) error
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
