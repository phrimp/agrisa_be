package services

import (
	utils "agrisa_utils"
	"policy-service/internal/models"
	"policy-service/internal/repository"
)

type BasePolicyTriggerService struct {
	BasePolicyTriggerRepository *repository.BasePolicyTriggerRepository
}

func NewBasePolicyTriggerService(basePolicyTriggerRepo *repository.BasePolicyTriggerRepository) *BasePolicyTriggerService {
	return &BasePolicyTriggerService{
		BasePolicyTriggerRepository: basePolicyTriggerRepo,
	}
}

func (s *BasePolicyTriggerService) GetBasePolicyTriggersByFilter(conditions []utils.Condition, orderBy []string) ([]models.BasePolicyTrigger, error) {
	templateQuery := "SELECT * FROM base_policy_triggers"
	return s.BasePolicyTriggerRepository.GetBasePolicyTriggerByFilter(conditions, orderBy, templateQuery)
}
