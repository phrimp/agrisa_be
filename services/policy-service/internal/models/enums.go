package models

type DataSourceType string

const (
	DataSourceWeather   DataSourceType = "weather"
	DataSourceSatellite DataSourceType = "satellite"
	DataSourceDerived   DataSourceType = "derived"
)

type ParameterType string

const (
	ParameterNumeric     ParameterType = "numeric"
	ParameterBoolean     ParameterType = "boolean"
	ParameterCategorical ParameterType = "categorical"
)

type BasePolicyStatus string

const (
	BasePolicyDraft    BasePolicyStatus = "draft"
	BasePolicyActive   BasePolicyStatus = "active"
	BasePolicyClosed   BasePolicyStatus = "closed"
	BasePolicyArchived BasePolicyStatus = "archived"
)

type PolicyStatus string

const (
	PolicyDraft          PolicyStatus = "draft"
	PolicyPendingReview  PolicyStatus = "pending_review"
	PolicyPendingPayment PolicyStatus = "pending_payment"
	PolicyActive         PolicyStatus = "active"
	PolicyPayout         PolicyStatus = "payout"
	PolicyExpired        PolicyStatus = "expired"
	PolicyPendingCancel  PolicyStatus = "pending_cancel"
	PolicyCancelled      PolicyStatus = "cancelled"
	PolicyRejected       PolicyStatus = "rejected"
)

type UnderwritingStatus string

const (
	UnderwritingPending  UnderwritingStatus = "pending"
	UnderwritingApproved UnderwritingStatus = "approved"
	UnderwritingRejected UnderwritingStatus = "rejected"
)

type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentPaid      PaymentStatus = "paid"
	PaymentOverdue   PaymentStatus = "overdue"
	PaymentCancelled PaymentStatus = "cancelled"
	PaymentRefunded  PaymentStatus = "refunded"
)

type PaymentType string

const (
	PaymentTypePolicyRegistration PaymentType = "policy_registration_payment"
	PaymentTypePolicyPayout       PaymentType = "policy_payout_payment"
	PaymentTypePolicyCompensation PaymentType = "policy_compensation_payment"
	PaymentTypePolicyRenewal      PaymentType = "policy_renewal_payment"
)

type ValidationStatus string

const (
	ValidationPending  ValidationStatus = "pending"
	ValidationPassed   ValidationStatus = "passed"
	ValidationPassedAI ValidationStatus = "passed_ai"
	ValidationFailed   ValidationStatus = "failed"
	ValidationWarning  ValidationStatus = "warning"
)

type ThresholdOperator string

const (
	ThresholdLT       ThresholdOperator = "<"
	ThresholdGT       ThresholdOperator = ">"
	ThresholdLTE      ThresholdOperator = "<="
	ThresholdGTE      ThresholdOperator = ">="
	ThresholdEQ       ThresholdOperator = "=="
	ThresholdNE       ThresholdOperator = "!="
	ThresholdChangeGT ThresholdOperator = "change_gt"
	ThresholdChangeLT ThresholdOperator = "change_lt"
)

type AggregationFunction string

const (
	AggregationSum    AggregationFunction = "sum"
	AggregationAvg    AggregationFunction = "avg"
	AggregationMin    AggregationFunction = "min"
	AggregationMax    AggregationFunction = "max"
	AggregationChange AggregationFunction = "change"
)

type LogicalOperator string

const (
	LogicalAND LogicalOperator = "AND"
	LogicalOR  LogicalOperator = "OR"
)

type ClaimStatus string

const (
	ClaimGenerated            ClaimStatus = "generated"
	ClaimPendingPartnerReview ClaimStatus = "pending_partner_review"
	ClaimApproved             ClaimStatus = "approved"
	ClaimRejected             ClaimStatus = "rejected"
	ClaimPaid                 ClaimStatus = "paid"
)

type PayoutStatus string

const (
	PayoutPending    PayoutStatus = "pending"
	PayoutProcessing PayoutStatus = "processing"
	PayoutCompleted  PayoutStatus = "completed"
	PayoutFailed     PayoutStatus = "failed"
)

type DataQuality string

const (
	DataQualityGood       DataQuality = "good"
	DataQualityAcceptable DataQuality = "acceptable"
	DataQualityPoor       DataQuality = "poor"
)

type FarmStatus string

const (
	FarmActive   FarmStatus = "active"
	FarmInactive FarmStatus = "inactive"
	FarmArchived FarmStatus = "archived"
)

type PhotoType string

const (
	PhotoCrop            PhotoType = "crop"
	PhotoBoundary        PhotoType = "boundary"
	PhotoLandCertificate PhotoType = "land_certificate"
	PhotoSatellite       PhotoType = "satellite"
	PhotoOther           PhotoType = "other"
)

type MonitorFrequency string

const (
	MonitorFrequencyHour  MonitorFrequency = "hour"
	MonitorFrequencyDay   MonitorFrequency = "day"
	MonitorFrequencyWeek  MonitorFrequency = "week"
	MonitorFrequencyMonth MonitorFrequency = "month"
	MonitorFrequencyYear  MonitorFrequency = "year"
)

type CancelRequestType string

const (
	CancelRequestTypeContractViolation   CancelRequestType = "contract_violation"
	CancelRequestTypeOther               CancelRequestType = "other"
	CancelRequestTypeNonPayment          CancelRequestType = "non_payment"
	CancelRequestTypePolicyholderRequest CancelRequestType = "policyholder_request"
	CancelRequestTypeRegulatoryChange    CancelRequestType = "regulatory_change"
)

type CancelRequestStatus string

const (
	CancelRequestStatusApproved      CancelRequestStatus = "approved"
	CancelRequestStatusLitigation    CancelRequestStatus = "litigation"
	CancelRequestStatusDenied        CancelRequestStatus = "denied"
	CancelRequestStatusPendingReview CancelRequestStatus = "pending_review"
)

type ClaimRejectionType string

const (
	ClaimRejectionTypeClaimDataIncorrect ClaimRejectionType = "claim_data_incorrect"
	ClaimRejectionTypeTriggerNotMet      ClaimRejectionType = "trigger_not_met"
	ClaimRejectionTypePolicyNotActive    ClaimRejectionType = "policy_not_active"
	ClaimRejectionTypeLocationMismatch   ClaimRejectionType = "location_mismatch"
	ClaimRejectionTypeDuplicateClaim     ClaimRejectionType = "duplicate_claim"
	ClaimRejectionTypeSuspectedFraud     ClaimRejectionType = "suspected_fraud"
	ClaimRejectionTypeOther              ClaimRejectionType = "other"
)

type DataSourceAPIAddress string

const (
	SatelliteNDVI   DataSourceAPIAddress = "/satellite/public/ndvi/batch"
	SatelliteNDMI   DataSourceAPIAddress = "/satellite/public/ndmi/batch"
	WeatherRainFall DataSourceAPIAddress = "/weather/public/api/v2/precipitation/polygon"
)

type DataSourceParameterName string

const (
	NDVI     DataSourceParameterName = "ndvi"
	NDMI     DataSourceParameterName = "ndmi"
	RainFall DataSourceParameterName = "rainfall"
)

type RiskAnalysisType string

const (
	RiskAnalysisTypeAIModel            RiskAnalysisType = "ai_model"
	RiskAnalysisTypeDocumentValidation RiskAnalysisType = "document_validation"
	RiskAnalysisTypeCrossReference     RiskAnalysisType = "cross_reference"
	RiskAnalysisTypeManual             RiskAnalysisType = "manual"
)

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)
