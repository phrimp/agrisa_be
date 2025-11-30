package repository

import (
	"auth-service/internal/models"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type IUserRepository interface {
	CreateUser(user *models.User) error
	GetUserByID(id string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByPhone(phone string) (*models.User, error)
	GetAllUsers(limit, offset int) ([]*models.User, error)
	GetUsersByStatus(status string) ([]*models.User, error)
	UpdateUser(user *models.User) error
	UpdatePassword(userID, newPassword string) error
	VerifyEmail(userID string) error
	VerifyPhone(userID string) error
	DeleteUser(userID string) error
	SoftDeleteUser(userID string) error
	UpdateUserNationalID(userID string, nationalID string) error
	UpdateUserFaceLiveness(userID string, faceLiveness string) error
	CheckPasswordHash(password, hash string) bool
	UpdateUserKycStatus(userID string, kycVerified bool) error
	UpdateUserStatus(userID string, status models.UserStatus, lockedUntil *int64) error
	CheckExistEmailOrPhone(value string) (bool, error)
	ResetEkycData(userID string) error
}

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) IUserRepository {
	return &UserRepository{
		db: db,
	}
}

func (u *UserRepository) CreateUser(user *models.User) error {
	hashedPassword, err := u.hashPassword(user.PasswordHash)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	query := `
		INSERT INTO users (id, phone_number, email, password_hash, national_id, status, 
		                  email_verified, phone_verified, kyc_verified, created_at, updated_at, locked_until, face_liveness)
		VALUES (:id, :phone_number, :email, :password_hash, :national_id, :status,
		        :email_verified, :phone_verified, :kyc_verified, :created_at, :updated_at, :locked_until, :face_liveness)
	`

	user.PasswordHash = hashedPassword
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err = u.db.NamedExec(query, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetUserByID(id string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE id = $1`

	err := r.db.Get(&user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE email = $1`

	err := r.db.Get(&user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetUserByPhone(phone string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE phone_number = $1`

	err := r.db.Get(&user, query, phone)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by phone: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetAllUsers(limit, offset int) ([]*models.User, error) {
	var users []*models.User
	query := `SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	err := r.db.Select(&users, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	return users, nil
}

func (r *UserRepository) GetUsersByStatus(status string) ([]*models.User, error) {
	var users []*models.User
	query := `SELECT * FROM users WHERE status = $1 ORDER BY created_at DESC`

	err := r.db.Select(&users, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by status: %w", err)
	}

	return users, nil
}

func (r *UserRepository) UpdateUser(user *models.User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users 
		SET phone_number = :phone_number, email = :email, national_id = :national_id,
		    status = :status, email_verified = :email_verified, phone_verified = :phone_verified,
		    kyc_verified = :kyc_verified, updated_at = :updated_at, last_login = :last_login,
		    login_attempts = :login_attempts, locked_until = :locked_until
		WHERE id = :id
	`

	result, err := r.db.NamedExec(query, user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *UserRepository) UpdatePassword(userID, newPassword string) error {
	hashedPassword, err := r.hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Exec(query, hashedPassword, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *UserRepository) VerifyEmail(userID string) error {
	query := `UPDATE users SET email_verified = true, updated_at = $1 WHERE id = $2`

	result, err := r.db.Exec(query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *UserRepository) VerifyPhone(userID string) error {
	query := `UPDATE users SET phone_verified = true, updated_at = $1 WHERE id = $2`

	result, err := r.db.Exec(query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to verify phone: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *UserRepository) DeleteUser(userID string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *UserRepository) SoftDeleteUser(userID string) error {
	query := `UPDATE users SET status = 'deactivated', updated_at = $1 WHERE id = $2`

	result, err := r.db.Exec(query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *UserRepository) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (r *UserRepository) CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (r *UserRepository) UpdateUserNationalID(userID string, nationalID string) error {
	query := `
		UPDATE users
		SET national_id = $1,
		    updated_at = $2
		WHERE id = $3
	`
	result, err := r.db.Exec(query, nationalID, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update national_id for user %s: %w", userID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no user found with id %s", userID)
	}

	return nil
}

func (r *UserRepository) UpdateUserFaceLiveness(userID string, faceLiveness string) error {
	query := `
		UPDATE users
		SET face_liveness = $1,
		    updated_at = now()
		WHERE id = $2
	`
	result, err := r.db.Exec(query, faceLiveness, userID)
	if err != nil {
		return fmt.Errorf("failed to update face_liveness for user %s: %w", userID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no user found with id %s", userID)
	}

	return nil
}

func (r *UserRepository) UpdateUserKycStatus(userID string, kycVerified bool) error {
	query := `
		UPDATE users
		SET kyc_verified = $1,
		    updated_at = NOW()
		WHERE id = $2
	`
	result, err := r.db.Exec(query, kycVerified, userID)
	if err != nil {
		return fmt.Errorf("failed to update kyc_verified for user %s: %w", userID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no user found with id %s", userID)
	}

	return nil
}

// UpdateUserStatus updates user status and optional locked_until timestamp
func (r *UserRepository) UpdateUserStatus(userID string, status models.UserStatus, lockedUntil *int64) error {
	query := `
		UPDATE users
		SET status = $1,
		    locked_until = $2,
		    updated_at = NOW()
		WHERE id = $3
	`
	result, err := r.db.Exec(query, status, lockedUntil, userID)
	if err != nil {
		return fmt.Errorf("failed to update status for user %s: %w", userID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no user found with id %s", userID)
	}

	return nil
}

func (r *UserRepository) CheckExistEmailOrPhone(value string) (bool, error) {
	var exists bool
	query := `
        SELECT EXISTS(
            SELECT 1 FROM users 
            WHERE email = $1 OR phone_number = $1
        )
    `
	err := r.db.Get(&exists, query, value)
	if err != nil {
		slog.Error("Error checking existence of email or phone", "error", err)
		return false, fmt.Errorf("failed to check existence of email or phone: %w", err)
	}
	return exists, nil
}

func (r *UserRepository) ResetEkycData(userID string) error {
	// Begin transaction
	tx, err := r.db.Beginx()
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure rollback if anything goes wrong
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Query 1: Update users table
	updateUsersQuery := `
        UPDATE users
        SET kyc_verified = false,
            face_liveness = '',
            updated_at = NOW()
        WHERE id = $1
    `
	result, err := tx.Exec(updateUsersQuery, userID)
	if err != nil {
		slog.Error("Failed to update users table", "error", err)
		return fmt.Errorf("failed to update users table: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		slog.Error("Failed to check rows affected on users", "error", err)
		return fmt.Errorf("failed to check rows affected on users: %w", err)
	}
	if rowsAffected == 0 {
		slog.Error("No user found with id", "userID", userID)
		return fmt.Errorf("user not found with id: %s", userID)
	}

	// Query 2: Update user_ekyc_progress table
	updateEkycProgressQuery := `
        UPDATE user_ekyc_progress
        SET is_ocr_done = false,
            ocr_done_at = NULL,
            is_face_verified = false,
            face_verified_at = NULL,
            cic_no = ''
        WHERE user_id = $1
    `
	result, err = tx.Exec(updateEkycProgressQuery, userID)
	if err != nil {
		slog.Error("Failed to update user_ekyc_progress table", "error", err)
		return fmt.Errorf("failed to update user_ekyc_progress table: %w", err)
	}

	rowsAffected, err = result.RowsAffected()
	if err != nil {
		slog.Error("Failed to check rows affected on user_ekyc_progress", "error", err)
		return fmt.Errorf("failed to check rows affected on user_ekyc_progress: %w", err)
	}
	if rowsAffected == 0 {
		slog.Error("No ekyc progress found for user", "userID", userID)
		return fmt.Errorf("user_ekyc_progress not found for user_id: %s", userID)
	}

	// Query 3: Delete from user_card table
	deleteUserCardQuery := `
        DELETE FROM user_card
        WHERE user_id = $1
    `
	_, err = tx.Exec(deleteUserCardQuery, userID)
	if err != nil {
		slog.Error("Failed to delete from user_card table", "error", err)
		return fmt.Errorf("failed to delete from user_card table: %w", err)
	}
	// Note: No need to check rowsAffected for delete, user may not have cards yet

	// Commit transaction
	if err = tx.Commit(); err != nil {
		slog.Error("Failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
