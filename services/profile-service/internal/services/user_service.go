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
}

func NewUserService(repo repository.IUserRepository) IUserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) GetUserProfileByUserID(userID string) (*models.UserProfile, error) {
	return s.repo.GetUserProfileByUserID(userID)
}