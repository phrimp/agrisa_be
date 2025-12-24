package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"profile-service/internal/event"
	"profile-service/internal/models"
	"profile-service/internal/repository"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
	"utils"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type InsurancePartnerService struct {
	repo                  repository.IInsurancePartnerRepository
	userProfileRepository repository.IUserRepository
	profilePublisher      *event.NotificationPublisher
}

type IInsurancePartnerService interface {
	GetPublicProfile(partnerID string) (*models.PublicPartnerProfile, error)
	GetPartnerReviews(partnerID string, sortBy string, sortDirection string, limit int, offset int) ([]models.PartnerReview, error)
	CreateInsurancePartner(req *models.CreateInsurancePartnerRequest, userID string) CreateInsurancePartnerResult
	GetPrivateProfile(userID string) (*models.PrivatePartnerProfile, error)
	UpdateInsurancePartner(updateProfileRequestBody map[string]any, updateByID, updateByName string) (*models.PrivatePartnerProfile, error)
	GetAllPartnersPublicProfiles() ([]models.PublicPartnerProfile, error)
	GetAllPartnersPrivateProfiles() ([]models.PrivatePartnerProfile, error)
	GetPrivateProfileByPartnerID(partnerID string) (*models.PrivatePartnerProfile, error)
	CreatePartnerDeletionRequest(req *models.PartnerDeletionRequest, partnerAdminID string) (result *models.PartnerDeletionRequest, err error)
	GetDeletionRequestsByRequesterID(requesterID string) ([]models.DeletionRequestResponse, error)
	ValidateDeletionRequestProcess(request models.ProcessRequestReviewDTO) (existDeletionRequest *models.DeletionRequestResponse, err error)
	ProcessRequestReviewByAdmin(request models.ProcessRequestReviewDTO, contracts map[string]any) error
	RevokePartnerDeletionRequest(requestID uuid.UUID, userID string, reviewNote string) error
	GetAllPartnerDeletionRequests() ([]models.DeletionRequestResponse, error)
	GetDeletionRequestsByPartnerID(partnerID, status string) ([]models.DeletionRequestResponse, error)
	GetPartnerDeletionRequestByID(requestID uuid.UUID) (*models.DeletionRequestResponse, error)
	GetActiveContracts(token string) (map[string]any, error)
}

func NewInsurancePartnerService(repo repository.IInsurancePartnerRepository, userProfileRepository repository.IUserRepository, profilePublisher *event.NotificationPublisher) IInsurancePartnerService {
	return &InsurancePartnerService{
		repo:                  repo,
		userProfileRepository: userProfileRepository,
		profilePublisher:      profilePublisher,
	}
}

func (s *InsurancePartnerService) GetPartnerReviews(partnerID string, sortBy string, sortDirection string, limit int, offset int) ([]models.PartnerReview, error) {
	return s.repo.GetPartnerReviews(partnerID, sortBy, sortDirection, limit, offset)
}

type CreateInsurancePartnerResult struct {
	Data    any
	Message string
}

func (s *InsurancePartnerService) CreateInsurancePartner(req *models.CreateInsurancePartnerRequest, userID string) CreateInsurancePartnerResult {
	// Trim all field values
	// trimmedDTO := utils.TrimAllStringFields(req).(*models.CreateInsurancePartnerRequest)

	// Validate
	validationErrors := ValidateInsurancePartner(req)
	if len(validationErrors) > 0 {
		// Handle validation errors
		validationErrorResponse := CreateInsurancePartnerResult{
			Data:    validationErrors,
			Message: "Validation errors occurred",
		}
		return validationErrorResponse
	}

	err := s.repo.CreateInsurancePartner(*req, userID, "Admin")
	if err != nil {
		// Handle creation error
		internalErrorResponse := CreateInsurancePartnerResult{
			Data:    err,
			Message: "Internal server error occurred",
		}
		return internalErrorResponse
	}

	successResponse := CreateInsurancePartnerResult{
		Data:    req,
		Message: "Success",
	}
	return successResponse
}

func ValidateInsurancePartner(req *models.CreateInsurancePartnerRequest) []*utils.ValidationError {
	var validationErrors []*utils.ValidationError
	// Validate Legal Company Name
	if err := ValidateLegalCompanyName(req.LegalCompanyName); err != nil {
		validationErrors = append(validationErrors, err)
	}
	// Validate Partner Trading Name
	if err := ValidatePartnerTradingName(req.PartnerTradingName); err != nil {
		validationErrors = append(validationErrors, err)
	}
	// Validate Partner Display Name
	if err := ValidatePartnerDisplayName(req.PartnerDisplayName); err != nil {
		validationErrors = append(validationErrors, err)
	}
	// Validate Company Type
	if err := ValidateCompanyType(req.CompanyType); err != nil {
		validationErrors = append(validationErrors, err)
	}
	// Validate Incorporation Date
	if err := ValidateIncorporationDate(&req.IncorporationDate, &req.YearEstablished, &req.LicenseIssueDate); err != nil {
		validationErrors = append(validationErrors, err)
	}
	// Validate Tax Identification Number
	if err := ValidateTaxIdentificationNumber(req.TaxIdentificationNumber); err != nil {
		validationErrors = append(validationErrors, err)
	}
	// Validate Business Registration Number
	if err := ValidateBusinessRegistrationNumber(req.BusinessRegistrationNumber); err != nil {
		validationErrors = append(validationErrors, err)
	}
	// Validate Partner Tagline
	if err := ValidatePartnerTagline(req.PartnerTagline); err != nil {
		validationErrors = append(validationErrors, err)
	}
	// // Validate Partner Phone
	// if err := ValidatePartnerPhone(req.PartnerPhone, "PartnerPhone"); err != nil {
	// 	validationErrors = append(validationErrors, err)
	// }
	// Validate Partner Official Email (required)
	if err := ValidatePartnerOfficialEmail(req.PartnerOfficialEmail, "PartnerOfficialEmail", true); err != nil {
		validationErrors = append(validationErrors, err)
	}

	// // Validate Head Office Address
	// if err := ValidateHeadOfficeAddress(req.HeadOfficeAddress); err != nil {
	// 	validationErrors = append(validationErrors, err)
	// }
	// // Validate Province Code
	// if err := ValidateProvinceCode(req.ProvinceCode); err != nil {
	// 	validationErrors = append(validationErrors, err)
	// }
	// // Validate Province Name
	// if err := ValidateProvinceName(req.ProvinceCode, req.ProvinceName); err != nil {
	// 	validationErrors = append(validationErrors, err)
	// }
	// // Validate Ward Code
	// if err := ValidateWardCode(req.ProvinceCode, req.WardCode); err != nil {
	// 	validationErrors = append(validationErrors, err)
	// }
	// // Validate Ward Name
	// if err := ValidateWardName(req.ProvinceCode, req.WardCode, req.WardName); err != nil {
	// 	validationErrors = append(validationErrors, err)
	// }
	// Validate Postal Code
	// if err := ValidatePostalCode(req.PostalCode); err != nil {
	// 	validationErrors = append(validationErrors, err)
	// }
	// Validate Insurance License Number
	if err := ValidateInsuranceLicenseNumber(req.InsuranceLicenseNumber); err != nil {
		validationErrors = append(validationErrors, err)
	}
	// Validate License Issue Date
	if err := ValidateLicenseIssueDate(req.LicenseIssueDate, req.IncorporationDate, req.LicenseExpiryDate); err != nil {
		validationErrors = append(validationErrors, err)
	}
	// Validate License Expiry Date
	if err := ValidateLicenseExpiryDate(req.LicenseExpiryDate, req.LicenseIssueDate); err != nil {
		validationErrors = append(validationErrors, err)
	}
	return validationErrors
}

var allowedUpdateInsuranceProfileFields = map[string]bool{
	"legal_company_name":           true,
	"partner_trading_name":         true,
	"partner_display_name":         true,
	"partner_logo_url":             true,
	"cover_photo_url":              true,
	"company_type":                 true,
	"incorporation_date":           true,
	"tax_identification_number":    true,
	"business_registration_number": true,
	"partner_tagline":              true,
	"partner_description":          true,
	"partner_phone":                true,
	"partner_official_email":       true,
	"head_office_address":          true,
	"province_code":                true,
	"province_name":                true,
	"ward_code":                    true,
	"ward_name":                    true,
	"postal_code":                  true,
	"fax_number":                   true,
	"customer_service_hotline":     true,
	"insurance_license_number":     true,
	"license_issue_date":           true,
	"license_expiry_date":          true,
	"authorized_insurance_lines":   true,
	"operating_provinces":          true,
	"year_established":             true,
	"partner_website":              true,
	"partner_rating_score":         true,
	"partner_rating_count":         true,
	"trust_metric_experience":      true,
	"trust_metric_clients":         true,
	"trust_metric_claim_rate":      true,
	"total_payouts":                true,
	"average_payout_time":          true,
	"confirmation_timeline":        true,
	"hotline":                      true,
	"support_hours":                true,
	"coverage_areas":               true,
	"status":                       true,
	"updated_at":                   true,
	"last_updated_by_id":           true,
	"last_updated_by_name":         true,
	"legal_document_urls":          true,
	"account_number":               true,
	"account_name":                 true,
	"bank_code":                    true,
}

var arrayInsuranceProfileFields = map[string]bool{
	"authorized_insurance_lines": true,
	"operating_provinces":        true,
	"legal_document_urls":        true,
}

func (s *InsurancePartnerService) UpdateInsurancePartner(updateProfileRequestBody map[string]any, updateByID, updateByName string) (*models.PrivatePartnerProfile, error) {
	// check if insurance partner profile exists
	var privateProfile *models.PrivatePartnerProfile
	partnerID := updateProfileRequestBody["partner_id"].(string)
	_, err := s.repo.GetInsurancePartnerByID(partnerID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			log.Printf("Insurance partner with ID %s does not exist", partnerID)
			return nil, fmt.Errorf("not found: insurance partner with ID %s does not exist", partnerID)
		}
		log.Printf("Error getting insurance partner by ID %s: %s", partnerID, err.Error())
		return nil, fmt.Errorf("internal server error: failed to get insurance partner: %v", err)
	}

	// Verify if the current user is authorized to update this profile
	updateUser, err := s.userProfileRepository.GetUserProfileByUserID(updateByID)
	if err != nil {
		log.Printf("Error getting user profile by ID %s: %s", updateByID, err.Error())
		return nil, fmt.Errorf("internal server error: failed to get user profile: %v", err)
	}

	if updateUser.PartnerID == nil {
		log.Printf("User ID %s is not associated with any partner", updateByID)
		return nil, fmt.Errorf("forbidden: user ID %s is not associated with any partner", updateByID)
	}

	if updateUser.PartnerID.String() != partnerID {
		log.Printf("User ID %s is not authorized to update partner ID %s", updateByID, partnerID)
		return nil, fmt.Errorf("forbidden: Bạn không có quyền cập nhật hồ sơ của đối tác bảo hiểm này")
	}

	delete(updateProfileRequestBody, "partner_id")
	updateProfileRequestBody["last_updated_by_id"] = updateByID
	updateProfileRequestBody["last_updated_by_name"] = updateByName

	// Build dynamic UPDATE query
	setClauses := []string{}
	args := []any{}
	argPosition := 1

	for field, value := range updateProfileRequestBody {
		// Kiểm tra field có được phép update không
		if !allowedUpdateInsuranceProfileFields[field] {
			log.Printf("Field %s is not allowed to be updated", field)
			return nil, fmt.Errorf("bad request: field %s is not allowed to be updated", field)
		}

		// Xử lý các field có kiểu array
		if arrayInsuranceProfileFields[field] {
			// Chuyển đổi slice any thành []string
			if arr, ok := value.([]any); ok {
				strArr := make([]string, len(arr))
				for i, v := range arr {
					strArr[i] = fmt.Sprintf("%v", v)
				}
				setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argPosition))
				args = append(args, pq.Array(strArr))
				argPosition++
			}
		} else {
			// Xử lý các field thông thường
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argPosition))
			args = append(args, value)
			argPosition++
		}
	}

	// Nếu không có field nào để update
	if len(setClauses) == 0 {
		log.Printf("No fields to update for insurance partner ID %s", partnerID)
		return nil, fmt.Errorf("bad request: no fields to update for insurance partner ID %s", partnerID)
	}

	hasUpdatedAt := false
	for field := range updateProfileRequestBody {
		if field == "updated_at" {
			hasUpdatedAt = true
			break
		}
	}
	if !hasUpdatedAt {
		setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argPosition))
		args = append(args, time.Now())
		argPosition++
	}

	args = append(args, partnerID)

	query := fmt.Sprintf(
		"UPDATE insurance_partners SET %s WHERE partner_id = $%d",
		strings.Join(setClauses, ", "),
		argPosition,
	)

	err = s.repo.UpdateInsurancePartner(query, args...)
	if err != nil {
		log.Printf("Error updating insurance partner ID %s: %s", partnerID, err.Error())
		return nil, fmt.Errorf("failed to update insurance partner: %v", err)
	}

	privateProfile, err = s.repo.GetPrivateProfile(partnerID)
	if err != nil {
		log.Printf("Error getting private profile for insurance partner ID %s after update: %s", partnerID, err.Error())
		return nil, fmt.Errorf("%v", err)
	}

	return privateProfile, nil
}

func ValidateLegalCompanyName(legalCompanyNameInput string) *utils.ValidationError {
	fieldName := "LegalCompanyName"

	// 1. Check if empty (Required)
	if legalCompanyNameInput == "" {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Tên công ty không được để trống",
		}
	}

	// 2. Check length (1-255 characters)
	length := utf8.RuneCountInString(legalCompanyNameInput)
	if length < 1 || length > 255 {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Tên công ty phải có độ dài từ 1 đến 255 ký tự",
		}
	}

	// 3. Check format (Vietnamese characters, spaces, no special characters)
	// Regex allows Vietnamese characters (with diacritics), English letters, and spaces
	vietnamesePattern := `^[a-zA-ZÀ-ỹ\s]+$`
	matched, err := regexp.MatchString(vietnamesePattern, legalCompanyNameInput)
	if err != nil || !matched {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Tên công ty chỉ được chứa chữ cái tiếng Việt và khoảng trắng, không chứa ký tự đặc biệt",
		}
	}

	// 4. Check business logic (Must start with company type prefix)
	// Common Vietnamese company prefixes
	validPrefixes := []string{
		"Công ty",
		"Công Ty",
		"CÔNG TY",
	}

	hasValidPrefix := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(legalCompanyNameInput, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Tên công ty phải bắt đầu bằng 'Công ty' theo quy định pháp luật Việt Nam",
		}
	}

	return nil
}

func ValidatePartnerTradingName(partnerTradingNameInput string) *utils.ValidationError {
	fieldName := "PartnerTradingName"

	length := utf8.RuneCountInString(partnerTradingNameInput)
	if length < 1 || length > 255 {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Tên giao dịch phải có độ dài từ 1 đến 255 ký tự",
		}
	}

	vietnamesePattern := `^[a-zA-ZÀ-ỹ\s]+$`
	matched, err := regexp.MatchString(vietnamesePattern, partnerTradingNameInput)
	if err != nil || !matched {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Tên giao dịch không được chứa ký tự đặc biệt",
		}
	}
	return nil
}

func ValidatePartnerDisplayName(partnerDisplayNameInput string) *utils.ValidationError {
	fieldName := "PartnerDisplayName"
	length := utf8.RuneCountInString(partnerDisplayNameInput)
	if length > 255 {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Tên hiển thị không được vượt quá 255 ký tự",
		}
	}

	vietnamesePattern := `^[a-zA-ZÀ-ỹ\s]+$`
	matched, err := regexp.MatchString(vietnamesePattern, partnerDisplayNameInput)
	if err != nil || !matched {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Tên hiển thị không được chứa ký tự đặc biệt",
		}
	}
	return nil
}

func ValidateCompanyType(companyTypeInput string) *utils.ValidationError {
	if companyTypeInput == "" {
		return nil
	}
	validCompanyTypes := map[string]bool{
		"domestic":      true,
		"foreign":       true,
		"joint_venture": true,
	}
	if !validCompanyTypes[companyTypeInput] {
		return &utils.ValidationError{
			Field:   "CompanyType",
			Message: "Company type must be one of: domestic, foreign, joint_venture",
		}
	}
	return nil
}

func ValidateIncorporationDate(incorporationDate *time.Time, yearEstablished *int, licenseIssueDate *time.Time) *utils.ValidationError {
	if incorporationDate == nil {
		return nil
	}

	// 1. Must not be a future date
	now := time.Now()
	if incorporationDate.After(now) {
		return &utils.ValidationError{
			Field:   "IncorporationDate",
			Message: "Incorporation date cannot be in the future",
		}
	}

	// 2. Business logic: Year must match year_established (if provided)
	if yearEstablished != nil {
		incorporationYear := incorporationDate.Year()
		log.Printf("incorporationDate value: %s", incorporationDate.String())
		if incorporationYear != *yearEstablished {
			log.Printf("Incorporation date year %d does not match year_established %d", incorporationYear, *yearEstablished)
			return &utils.ValidationError{
				Field:   "IncorporationDate",
				Message: "Incorporation date year must match year_established",
			}
		}
	}

	// 3. Business logic: Must be before license_issue_date (if provided)
	if licenseIssueDate != nil {
		if !incorporationDate.Before(*licenseIssueDate) {
			return &utils.ValidationError{
				Field:   "IncorporationDate",
				Message: "Incorporation date must be before license issue date",
			}
		}
	}

	return nil
}

// ValidateTaxIdentificationNumber validates the tax identification number according to Vietnamese regulations
func ValidateTaxIdentificationNumber(taxIdentificationNumberInput string) *utils.ValidationError {
	fieldName := "TaxIdentificationNumber"

	// 1. Check Required
	if taxIdentificationNumberInput == "" {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Mã số thuế không được để trống",
		}
	}

	// 2. Check Format: Only digits and optional "-" (regex: ^\d{10}(-\d{3})?$)
	formatRegex := regexp.MustCompile(`^\d{10}(-\d{3})?$`)
	if !formatRegex.MatchString(taxIdentificationNumberInput) {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Mã số thuế phải có định dạng 10 chữ số hoặc 10 chữ số theo sau bởi -XXX (13 ký tự)",
		}
	}

	// All validations passed
	return nil
}

func ValidateBusinessRegistrationNumber(businessRegistrationNumberInput string) *utils.ValidationError {
	fieldName := "BusinessRegistrationNumber"

	// 1. Check Required
	if businessRegistrationNumberInput == "" {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Mã số đăng ký kinh doanh không được để trống",
		}
	}

	// 2. Check Format: Only digits and optional "-" (regex: ^\d{10}(-\d{3})?$)
	formatRegex := regexp.MustCompile(`^\d{10}(-\d{3})?$`)
	if !formatRegex.MatchString(businessRegistrationNumberInput) {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Mã số đăng ký kinh doanh phải có định dạng 10 chữ số hoặc 10 chữ số theo sau bởi -XXX (13 ký tự)",
		}
	}

	// All validations passed
	return nil
}

func ValidatePartnerTagline(partnerTaglineInput string) *utils.ValidationError {
	fieldName := "PartnerTagline"

	// 1. Optional field - if empty or whitespace only, it's valid
	trimmed := strings.TrimSpace(partnerTaglineInput)
	if trimmed == "" {
		return nil
	}

	// 2. Check Max Length: 500 characters
	// Using utf8.RuneCountInString to count characters correctly for Vietnamese text
	charCount := utf8.RuneCountInString(trimmed)
	if charCount > 500 {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Slogan không được vượt quá 500 ký tự",
		}
	}

	// All validations passed
	return nil
}

func ValidatePartnerPhone(partnerPhoneInput string, fieldName string) *utils.ValidationError {
	// 1. Optional field - if empty or whitespace only, it's valid
	trimmed := strings.TrimSpace(partnerPhoneInput)
	if trimmed == "" {
		return nil
	}

	// 3. Check Format: Must follow Vietnamese phone format with +84 prefix
	// Pattern: +84 followed by 9 or 10 digits
	phoneRegex := regexp.MustCompile(`^0\d{9,10}$`)
	if !phoneRegex.MatchString(trimmed) {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Số điện thoại/fax phải có số 0 đầu tiên và theo sau bởi 9 chữ số (ví dụ: 0865921357)",
		}
	}

	// All validations passed
	return nil
}

func ValidatePartnerOfficialEmail(emailInput string, fieldName string, isRequired bool) *utils.ValidationError {
	// Use default field name if not provided
	if fieldName == "" {
		fieldName = "Email"
	}

	trimmed := strings.TrimSpace(emailInput)

	// 1. Check Required (if applicable)
	if isRequired && trimmed == "" {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Email không được để trống",
		}
	}

	// 2. If optional and empty, it's valid
	if !isRequired && trimmed == "" {
		return nil
	}

	// 3. Check Max Length (common limit: 254 characters per RFC 5321)
	if len(trimmed) > 254 {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Email không được vượt quá 254 ký tự",
		}
	}

	// 4. Check Min Length (reasonable minimum: a@b.c = 5 characters)
	if len(trimmed) < 5 {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Email phải có ít nhất 5 ký tự",
		}
	}

	// 5. Check Format: Standard email regex pattern
	// This pattern covers most valid email formats
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(trimmed) {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Email không đúng định dạng (ví dụ: example@domain.com)",
		}
	}

	// 6. Additional validations

	// Check for multiple @ symbols
	if strings.Count(trimmed, "@") != 1 {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Email chỉ được chứa một ký tự @",
		}
	}

	// Split email into local part and domain
	parts := strings.Split(trimmed, "@")
	localPart := parts[0]
	domain := parts[1]

	// Check local part length (max 64 characters per RFC 5321)
	if len(localPart) > 64 {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Phần tên email (trước @) không được vượt quá 64 ký tự",
		}
	}

	// Check domain part length (max 255 characters per RFC 5321)
	if len(domain) > 255 {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Phần domain (sau @) không được vượt quá 255 ký tự",
		}
	}

	// Check for consecutive dots
	if strings.Contains(trimmed, "..") {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Email không được chứa hai dấu chấm liên tiếp",
		}
	}

	// Check for leading/trailing dots in local part
	if strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Phần tên email không được bắt đầu hoặc kết thúc bằng dấu chấm",
		}
	}

	// Check for leading/trailing hyphens in domain
	if strings.HasPrefix(domain, "-") || strings.HasSuffix(domain, "-") {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Domain không được bắt đầu hoặc kết thúc bằng dấu gạch ngang",
		}
	}

	// All validations passed
	return nil
}

func ValidateHeadOfficeAddress(headOfficeAddressInput string) *utils.ValidationError {
	fieldName := "HeadOfficeAddress"

	// 1. Optional field - if empty or whitespace only, it's valid
	trimmed := strings.TrimSpace(headOfficeAddressInput)
	if trimmed == "" {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Địa chỉ trụ sở chính không được để trống",
		}
	}

	// 2. Check Max Length (common limit: 255 characters)
	if len(trimmed) > 255 {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Địa chỉ trụ sở chính không được vượt quá 255 ký tự",
		}
	}

	// All validations passed
	return nil
}

func ValidateProvinceCode(provinceCode string) *utils.ValidationError {
	// Step 1: Check if province code is empty
	fieldName := "ProvinceCode"
	trimmed := strings.TrimSpace(provinceCode)
	if trimmed == "" {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Mã tỉnh không được để trống",
		}
	}

	// Step 2: Call the Province API
	provinces, err := utils.GetProvinceInfo()
	if err != nil {
		log.Printf("Error fetching provinces from API: %s", err.Error())
		return &utils.ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("Đã có lỗi xảy ra"),
		}
	}

	// Step 3: Check if the province code exists in the response
	isValid := false
	for _, province := range provinces {
		if province.Code == trimmed {
			isValid = true
			break
		}
	}

	if !isValid {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Mã tỉnh không hợp lệ",
		}
	}
	return nil
}

func ValidateProvinceName(provinceCodeInput string, provinceNameInput string) *utils.ValidationError {
	// Step 1: Check if province name is empty
	fieldName := "ProvinceName"
	trimmed := strings.TrimSpace(provinceNameInput)
	if trimmed == "" {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Tên tỉnh/thành phố không được để trống",
		}
	}

	// Step 2: Get province information from API
	provinces, err := utils.GetProvinceInfo()
	if err != nil {
		log.Printf("Error fetching provinces from API: %s", err.Error())
		return &utils.ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("Đã có lỗi xảy ra"),
		}
	}

	// Step 3: Find the province with matching code and compare name
	trimmedCode := strings.TrimSpace(provinceCodeInput)
	provinceFound := false

	for _, province := range provinces {
		if province.Code == trimmedCode {
			provinceFound = true
			// Compare the province name
			if province.Name != trimmed {
				return &utils.ValidationError{
					Field:   fieldName,
					Message: "Tên tỉnh/thành phố không khớp với mã tỉnh/thành phố",
				}
			}
			// Name matches, validation passed
			return nil
		}
	}

	// If province code not found, return error
	if !provinceFound {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Mã tỉnh/thành phố không hợp lệ",
		}
	}

	return nil
}

func ValidateWardCode(provinceCodeInput string, wardCodeInput string) *utils.ValidationError {
	// Step 1: Check if ward code is empty
	fieldName := "WardCode"
	trimmed := strings.TrimSpace(wardCodeInput)
	if trimmed == "" {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Mã phường/xã không được để trống",
		}
	}

	// Step 2: Call the Ward/Commune API
	communes, err := utils.GetWardInfo(provinceCodeInput)
	if err != nil {
		log.Printf("Error fetching communes from API: %s", err.Error())
		return &utils.ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("Đã có lỗi xảy ra khi lấy thông tin phường xã"),
		}
	}

	// Step 3: Check if the ward code exists in the response
	isValid := false
	for _, commune := range communes {
		if commune.Code == trimmed {
			isValid = true
			break
		}
	}

	if !isValid {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Mã phường/xã không hợp lệ",
		}
	}

	// All validations passed
	return nil
}

func ValidateWardName(provinceCodeInput string, wardCodeInput string, wardNameInput string) *utils.ValidationError {
	// Step 1: Check if ward name is empty
	fieldName := "WardName"
	trimmed := strings.TrimSpace(wardNameInput)
	if trimmed == "" {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Tên phường/xã không được để trống",
		}
	}

	// Step 2: Get ward information from API
	communes, err := utils.GetWardInfo(provinceCodeInput)
	if err != nil {
		log.Printf("Error fetching communes from API: %s", err.Error())
		return &utils.ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("Đã có lỗi xảy ra khi lấy thông tin phường xã"),
		}
	}

	// Step 3: Find the ward with matching code and compare name
	trimmedCode := strings.TrimSpace(wardCodeInput)
	wardFound := false

	for _, commune := range communes {
		if commune.Code == trimmedCode {
			wardFound = true
			// Compare the ward name
			if commune.Name != trimmed {
				return &utils.ValidationError{
					Field:   fieldName,
					Message: "Tên phường/xã không khớp với mã phường/xã",
				}
			}
			// Name matches, validation passed
			return nil
		}
	}

	// If ward code not found, return error
	if !wardFound {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Mã phường/xã không hợp lệ",
		}
	}

	return nil
}

// ValidatePostalCode validates the postal code format
func ValidatePostalCode(postalCodeInput string) *utils.ValidationError {
	fieldName := "PostalCode"

	// Step 1: Optional field - if empty, it's valid
	trimmed := strings.TrimSpace(postalCodeInput)
	if trimmed == "" {
		return nil
	}

	// Step 2: Check format - must be 5 or 6 digits
	postalCodeRegex := regexp.MustCompile(`^\d{5,6}$`)
	if !postalCodeRegex.MatchString(trimmed) {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Postal code không hợp lệ",
		}
	}

	// Step 3: All validations passed
	return nil
}

func ValidateInsuranceLicenseNumber(insuranceLicenseNumberInput string) *utils.ValidationError {
	fieldName := "InsuranceLicenseNumber"

	// 1. Check Required
	if insuranceLicenseNumberInput == "" {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Số giấy phép bảo hiểm không được để trống",
		}
	}

	// 2. Check Format: Only digits and optional "-" (regex: ^\d{10}(-\d{3})?$)
	formatRegex := regexp.MustCompile(`^\d{10}(-\d{3})?$`)
	if !formatRegex.MatchString(insuranceLicenseNumberInput) {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Số giấy phép bảo hiểm phải có định dạng 10 chữ số hoặc 10 chữ số theo sau bởi -XXX (13 ký tự)",
		}
	}

	// All validations passed
	return nil
}

func ValidateLicenseIssueDate(licenseIssueDate time.Time, incorporationDate time.Time, licenseExpiryDate time.Time) *utils.ValidationError {
	fieldName := "LicenseIssueDate"

	if licenseIssueDate.IsZero() {
		return nil
	}

	if !incorporationDate.IsZero() && licenseIssueDate.Before(incorporationDate) {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Ngày cấp giấy phép phải sau ngày thành lập công ty",
		}
	}

	if !licenseExpiryDate.IsZero() && licenseIssueDate.After(licenseExpiryDate) {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Ngày cấp giấy phép phải trước ngày hết hạn giấy phép",
		}
	}

	// Step 5: Must be before or equal to current date
	currentDate := time.Now()
	if licenseIssueDate.After(currentDate) {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Ngày cấp giấy phép không được là ngày tương lai",
		}
	}

	// All validations passed
	return nil
}

func ValidateLicenseExpiryDate(licenseExpiryDate time.Time, licenseIssueDate time.Time) *utils.ValidationError {
	fieldName := "LicenseExpiryDate"

	if licenseExpiryDate.IsZero() {
		return nil
	}

	if !licenseIssueDate.IsZero() && licenseExpiryDate.Before(licenseIssueDate) {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Ngày hết hạn giấy phép phải sau ngày cấp giấy phép",
		}
	}

	currentDate := time.Now()
	expiryDateOnly := time.Date(licenseExpiryDate.Year(), licenseExpiryDate.Month(), licenseExpiryDate.Day(), 0, 0, 0, 0, licenseExpiryDate.Location())
	currentDateOnly := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, currentDate.Location())

	if expiryDateOnly.Before(currentDateOnly) {
		return &utils.ValidationError{
			Field:   fieldName,
			Message: "Giấy phép đã hết hạn",
		}
	}

	// All validations passed
	return nil
}

func (s *InsurancePartnerService) GetPublicProfile(partnerID string) (*models.PublicPartnerProfile, error) {
	return s.repo.GetPublicProfile(partnerID)
}

func (s *InsurancePartnerService) GetAllPartnersPublicProfiles() ([]models.PublicPartnerProfile, error) {
	return s.repo.GetAllPublicProfiles()
}

func (s *InsurancePartnerService) GetAllPartnersPrivateProfiles() ([]models.PrivatePartnerProfile, error) {
	return s.repo.GetAllPrivateProfiles()
}

func (s *InsurancePartnerService) GetPrivateProfile(userID string) (*models.PrivatePartnerProfile, error) {
	staff, err := s.userProfileRepository.GetUserProfileByUserID(userID)
	if err != nil {
		return nil, err
	}
	if (staff.PartnerID == nil) || (staff.PartnerID.String() == "") {
		return nil, fmt.Errorf("forbidden: user is not associated with any insurance partner")
	}
	partnerID := staff.PartnerID
	return s.repo.GetPrivateProfile(partnerID.String())
}

func (s *InsurancePartnerService) GetPrivateProfileByPartnerID(partnerID string) (*models.PrivatePartnerProfile, error) {
	return s.repo.GetPrivateProfile(partnerID)
}

// ======= PARTNER DELETION REQUESTS =======
func (s *InsurancePartnerService) CreatePartnerDeletionRequest(req *models.PartnerDeletionRequest, partnerAdminID string) (result *models.PartnerDeletionRequest, err error) {
	partner, err := s.GetPrivateProfile(partnerAdminID)
	if err != nil {
		return nil, err
	}
	req.PartnerID = &partner.PartnerID

	userProfile, err := s.userProfileRepository.GetUserProfileByUserID(partnerAdminID)
	if err != nil {
		return nil, err
	}

	// Fetch the most recent request. If its status is pending, block new requests and notify the user that the previous one is still being processed.
	latestRequest, err := s.repo.GetLatestDeletionRequestByRequesterID(userProfile.UserID)
	if err != nil && !strings.Contains(err.Error(), "no rows in result set") {
		return nil, err
	}

	if latestRequest != nil && latestRequest.Status == models.DeletionRequestPending {
		slog.Error("A pending deletion request already exists for this user", "userID", userProfile.UserID)
		return nil, fmt.Errorf("invalid: a pending deletion request already exists. Please wait for it to be processed before submitting a new one")
	}

	req.RequestedBy = userProfile.UserID
	req.RequestedByName = userProfile.FullName
	req.Status = models.DeletionRequestPending
	req.RequestedAt = time.Now()
	// publish event
	go func() {
		eventPayload := event.ProfileEvent{
			ID:        uuid.NewString(),
			EventType: event.ProfleConfirmDelete,
			UserID:    req.RequestedBy,
			ProfileID: req.PartnerID.String(),
		}
		err := s.profilePublisher.PublishEvent(context.Background(), eventPayload)
		if err != nil {
			slog.Error("error publishing event", "error", err)
			return
		}
		slog.Info("profile event published", "event", eventPayload)
	}()

	return s.repo.CreateDeletionRequest(context.Background(), req)
}

func (s *InsurancePartnerService) GetDeletionRequestsByRequesterID(requesterID string) ([]models.DeletionRequestResponse, error) {
	return s.repo.GetDeletionRequestsByRequesterID(context.Background(), requesterID)
}

func (s *InsurancePartnerService) ProcessRequestReviewByAdmin(request models.ProcessRequestReviewDTO, contracts map[string]any) error {
	now := time.Now()
	adminID := request.ReviewedByID
	adminProfile, err := s.userProfileRepository.GetUserProfileByUserID(adminID)
	if err != nil {
		return err
	}

	request.ReviewedByID = adminProfile.UserID
	request.ReviewedByName = adminProfile.FullName

	// validate request
	existDeletionRequest, err := s.ValidateDeletionRequestProcess(request)
	if err != nil {
		return err
	}

	if request.Status == models.DeletionRequestApproved && len(contracts) > 0 {
		slog.Error("Cannot approve deletion request: active contracts exist", "requestID", request.RequestID)
		return fmt.Errorf("invalid: Không thể phê duyệt yêu cầu xóa hồ sơ đối tác bảo hiểm vì vẫn còn hợp đồng bảo hiểm đang hoạt động")
	}

	if request.Status == models.DeletionRequestApproved {
		// update status of partner profile
		noticePeriod := now.Add(models.NoticePeriod * time.Hour)
		// run cronj
		go func(noticePeriod time.Time, partnerID uuid.UUID, reviewedByID, reviewedByName string) {
			for {
				now := time.Now()
				if now.Compare(noticePeriod) == 0 {
					err = s.repo.UpdateStatusPartnerProfile(partnerID, "terminated", reviewedByID, reviewedByName, noticePeriod)
					if err != nil {
						// logging input values
						slog.Error("Failed to update partner profile status: err", "partnerID", partnerID, "status", "terminated", "updatedByID", reviewedByID, "updatedByName", reviewedByName, "error", err)
						slog.Error("Failed to update partner profile status: err", "error", err)
					}
					return
				}
				time.Sleep(1 * time.Second)
			}
		}(noticePeriod, existDeletionRequest.PartnerID, request.ReviewedByID, request.ReviewedByName)
	}
	return s.repo.ProcessRequestReview(request)
}

func (s *InsurancePartnerService) GetActiveContracts(token string) (map[string]any, error) {
	url := "http://policy-service:8089/policy/protected/api/v2/policies/read-partner/active"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("Error creating request for active contracts", "error", err)
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error making request for active contracts", "error", err)
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Error reading response body for active contracts", "error", err)
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		slog.Error("active contracts not found", "status_code", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("active contracts not found %d, body: %s", resp.StatusCode, string(body))
	}
	if resp.StatusCode == http.StatusForbidden {
		slog.Error("active contracts not found", "status_code", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("active contracts not found %d, body: %s", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("Unexpected status code for active contracts", "status_code", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error("Error parsing JSON for active contracts", "error", err)
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return result, nil
}

func (s *InsurancePartnerService) ValidateDeletionRequestProcess(request models.ProcessRequestReviewDTO) (existDeletionRequest *models.DeletionRequestResponse, err error) {
	// Validate Request ID
	if strings.TrimSpace(request.RequestID.String()) == "" {
		slog.Error("RequestID is required")
		return nil, fmt.Errorf("invalid: RequestID là bắt buộc")
	}

	// check if deletion request exists
	deletionRequest, err := s.repo.GetDeletionRequestsByRequestID(request.RequestID)
	if err != nil {
		return nil, err
	}

	// Validate Status
	if deletionRequest.Status != models.DeletionRequestPending {
		slog.Error("Only pending requests can be processed")
		return nil, fmt.Errorf("invalid: Chỉ các yêu cầu đang chờ xử lý mới có thể được xử lý")
	}

	return deletionRequest, nil
}

func (s *InsurancePartnerService) RevokePartnerDeletionRequest(requestID uuid.UUID, userID string, reviewNote string) error {
	// check if deletion request exists
	deletionRequest, err := s.repo.GetDeletionRequestsByRequestID(requestID)
	if err != nil {
		return err
	}

	// check if the requester is the same as the user trying to revoke
	if deletionRequest.RequestedBy != userID {
		slog.Error("User is not authorized to revoke this deletion request", "userID", userID, "requestedBy", deletionRequest.RequestedBy)
		return fmt.Errorf("forbidden: Bạn không có quyền hủy yêu cầu này")
	}

	// check if the request is still pending
	if deletionRequest.Status == models.DeletionRequestApproved {
		slog.Error("Approved requests cannot be revoked", "requestID", requestID, "status", deletionRequest.Status)
		return fmt.Errorf("invalid: Không thể thu hồi yêu cầu đã được phê duyệt ")
	}

	// check if now is before cancellable until
	//now := time.Now()
	//revokeAllow := models.NoticePeriod * 0.2
	//revokeAllowTime := now.Add((time.Duration(revokeAllow) * time.Hour))
	//if now.After(revokeAllowTime) {
	//	slog.Error("Cannot revoke request after cancellable until time", "currentTime", now, "cancellableUntil", deletionRequest.CancellableUntil)
	//	return fmt.Errorf("invalid: Không thể tự hủy yêu cầu sau thời gian đơn đã được duyệt 6 ngày")
	//}

	// get revoker profile
	revokerProfile, err := s.userProfileRepository.GetUserProfileByUserID(userID)
	if err != nil {
		return err
	}

	processRequestReview := models.ProcessRequestReviewDTO{
		RequestID:      requestID,
		Status:         models.DeletionRequestCancelled,
		ReviewedByID:   userID,
		ReviewedByName: revokerProfile.FullName,
		ReviewNote:     reviewNote,
	}

	go func() {
		eventPayload := event.ProfileEvent{
			ID:        uuid.NewString(),
			EventType: event.ProfileCancelDelete,
			UserID:    userID,
			ProfileID: deletionRequest.PartnerID.String(),
		}
		err := s.profilePublisher.PublishEvent(context.Background(), eventPayload)
		if err != nil {
			slog.Error("error publishing event", "error", err)
			return
		}
		slog.Info("profile event published", "event", eventPayload)
	}()

	return s.repo.ProcessRequestReview(processRequestReview)
}

func (s *InsurancePartnerService) GetAllPartnerDeletionRequests() ([]models.DeletionRequestResponse, error) {
	return s.repo.GetAllDeletionRequests(context.Background())
}

func (s *InsurancePartnerService) GetPartnerDeletionRequestByID(requestID uuid.UUID) (*models.DeletionRequestResponse, error) {
	return s.repo.GetDeletionRequestsByRequestID(requestID)
}

func (s *InsurancePartnerService) GetDeletionRequestsByPartnerID(partnerID, status string) ([]models.DeletionRequestResponse, error) {
	return s.repo.GetDeletionRequestsByPartnerID(context.Background(), partnerID, status)
}
