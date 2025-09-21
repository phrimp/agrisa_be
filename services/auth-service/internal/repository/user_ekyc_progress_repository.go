package repository

import (
	"auth-service/internal/models"
	"database/sql"
	"fmt"

	agrisa_utils "agrisa_utils"

	"github.com/jmoiron/sqlx"
)

type IUserEkycProgressRepository interface {
	UpdateOCRDone(userID string, ocrDone bool, nationalID string) error
	GetUserEkycProgressByUserID(userID string) (*models.UserEkycProgress, error)
	UpdateFaceLivenessDone(userID string, isFaceLivenessDone bool) error
	CreateUserEkycProgress(progress *models.UserEkycProgress) error
}

type UserEkycProgressRepository struct {
	db *sqlx.DB
}

func NewUserEkycProgressRepository(db *sqlx.DB) IUserEkycProgressRepository {
	return &UserEkycProgressRepository{
		db: db,
	}
}

func (r *UserEkycProgressRepository) GetUserEkycProgressByUserID(userID string) (*models.UserEkycProgress, error) {
	var progress models.UserEkycProgress
	query := `SELECT * FROM user_ekyc_progress WHERE user_id = $1`
	err := r.db.Get(&progress, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user ekyc progress not found")
		}
		return nil, fmt.Errorf("failed to get user ekyc progress by user ID: %w", err)
	}
	return &progress, nil
}

func (u *UserEkycProgressRepository) UpdateOCRDone(userID string, isOcrDone bool, nationalID string) error {
	query := `
		UPDATE user_ekyc_progress
		SET is_ocr_done = $1,
		    ocr_done_at = NOW(),
			cic_no = $3
		WHERE user_id = $2
	`

	result, err := u.db.Exec(query, isOcrDone, userID, nationalID)
	if err != nil {
		return fmt.Errorf("failed to update ocr_done: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows updated for user_id: %s", userID)
	}
	return nil
}

func (u *UserEkycProgressRepository) UpdateFaceLivenessDone(userID string, isFaceLivenessDone bool) error {
	query := `
		UPDATE user_ekyc_progress
		SET is_face_verified = $1,
		    face_verified_at = NOW()
		WHERE user_id = $2
	`

	result, err := u.db.Exec(query, isFaceLivenessDone, userID)
	if err != nil {
		return fmt.Errorf("failed to update face_liveness_done: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows updated for user_id: %s", userID)
	}
	return nil
}

func (u *UserEkycProgressRepository) CreateUserEkycProgress(progress *models.UserEkycProgress) error {
	query := `
		INSERT INTO user_ekyc_progress (
			user_id,
			cic_no,
			is_ocr_done,
			ocr_done_at,
			is_face_verified,
			face_verified_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	return agrisa_utils.ExecWithCheck(u.db, query, agrisa_utils.ExecInsert,
		progress.UserID,
		progress.CicNo,
		progress.IsOcrDone,
		progress.OcrDoneAt,
		progress.IsFaceVerified,
		progress.FaceVerifiedAt,
	)

}
