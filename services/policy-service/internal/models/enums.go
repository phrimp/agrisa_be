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
	BasePolicyArchived BasePolicyStatus = "archived"
)

type PolicyStatus string

const (
	PolicyDraft         PolicyStatus = "draft"
	PolicyPendingReview PolicyStatus = "pending_review"
	PolicyActive        PolicyStatus = "active"
	PolicyExpired       PolicyStatus = "expired"
	PolicyCancelled     PolicyStatus = "cancelled"
	PolicyRejected      PolicyStatus = "rejected"
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
	CancelRequestTypeContractViolation CancelRequestType = "contract_violation"
	CancelRequestTypeOther             CancelRequestType = "other"
)

type CancelRequestStatus string

const (
	CancelRequestStatusApproved   CancelRequestStatus = "approved"
	CancelRequestStatusLitigation CancelRequestStatus = "litigation"
	CancelRequestStatusDenied     CancelRequestStatus = "denied"
)

type ClaimRejectionType string

const (
	ClaimRejectionTypeClaimDataIncorrect ClaimRejectionType = "claim_data_incorrect"
)
