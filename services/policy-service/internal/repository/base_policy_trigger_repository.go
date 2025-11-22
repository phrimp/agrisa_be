package repository

import (
	utils "agrisa_utils"
	"log/slog"
	"policy-service/internal/models"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type BasePolicyTriggerRepository struct {
	db          *sqlx.DB
	redisClient *redis.Client
}

func NewBasePolicyTriggerRepository(db *sqlx.DB, redisClient *redis.Client) *BasePolicyTriggerRepository {
	return &BasePolicyTriggerRepository{
		db:          db,
		redisClient: redisClient,
	}
}

func (s *BasePolicyTriggerRepository) GetBasePolicyTriggerByFilter(conditions []utils.Condition, orderBy []string, templateQuery string) ([]models.BasePolicyTrigger, error) {
	qb := utils.QueryBuilder{
		TemplateQuery: templateQuery,
		Conditions:    conditions,
		OrderBy:       orderBy,
	}
	query, args, err := qb.BuildQueryDynamicFilter()
	if err != nil {
		slog.Error("Error building query for GetBasePolicyTriggerByFilter:", err)
		return nil, err
	}

	var results []models.BasePolicyTrigger
	err = s.db.Select(&results, query, args...)
	if err != nil {
		slog.Error("Error executing query for GetBasePolicyTriggerByFilter:", err)
		return nil, err
	}

	return results, nil
}
