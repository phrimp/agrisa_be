-- ============================================================================
-- AGRISA: Satellite-Powered Agricultural Insurance Platform
-- PostgreSQL Database Schema - Corrected Version
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";

-- ============================================================================
-- ENUM TYPES (same as before)
-- ============================================================================

CREATE TYPE data_source_type AS ENUM ('weather', 'satellite', 'derived');
CREATE TYPE parameter_type AS ENUM ('numeric', 'boolean', 'categorical');
CREATE TYPE base_policy_status AS ENUM ('draft', 'active', 'closed', 'archived');
CREATE TYPE policy_status AS ENUM ('draft', 'pending_review', 'pending_payment','payout', 'active', 'expired','pending_cancel', 'cancelled', 'rejected', 'dispute', 'cancelled_pending_payment');
CREATE TYPE underwriting_status AS ENUM ('pending', 'approved', 'rejected');
CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'overdue', 'cancelled', 'refunded');
CREATE TYPE validation_status AS ENUM ('pending', 'passed', 'passed_ai', 'failed', 'warning');
CREATE TYPE threshold_operator AS ENUM ('<', '>', '<=', '>=', '==', '!=', 'change_gt', 'change_lt');
CREATE TYPE aggregation_function AS ENUM ('sum', 'avg', 'min', 'max', 'change');
CREATE TYPE logical_operator AS ENUM ('AND', 'OR');
CREATE TYPE claim_status AS ENUM ('generated', 'pending_partner_review', 'approved', 'rejected', 'paid');
CREATE TYPE payout_status AS ENUM ('pending', 'processing', 'completed', 'failed');
CREATE TYPE data_quality AS ENUM ('good', 'acceptable', 'poor');
CREATE TYPE farm_status AS ENUM ('active', 'inactive', 'archived');
CREATE TYPE photo_type AS ENUM ('crop', 'boundary', 'land_certificate', 'other', 'satellite');
CREATE TYPE monitor_frequency AS ENUM ('hour', 'day', 'week', 'month', 'year');
CREATE TYPE cancel_request_type as ENUM ('contract_violation', 'other', 'non_payment', 'policyholder_request', 'regulatory_change');
CREATE TYPE cancel_request_status as ENUM ('approved', 'litigation', 'denied', 'pending_review', 'cancelled', 'payment_failed');
CREATE TYPE claim_rejection_type as ENUM ('claim_data_incorrect', 'trigger_not_met', 'policy_not_active', 'location_mismatch', 'duplicate_claim', 'suspected_fraud', 'other');
CREATE TYPE risk_analysis_type AS ENUM ('ai_model', 'document_validation', 'cross_reference', 'manual');
CREATE TYPE risk_level AS ENUM ('low', 'medium', 'high', 'critical');
-- ============================================================================
-- CORE DATA SOURCE & PRICING TABLES
-- ============================================================================

-- Category (Top level: Weather, Satellite, Derived)
CREATE TABLE data_tier_category (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category_name VARCHAR(100) NOT NULL UNIQUE,
    category_description TEXT,
    category_cost_multiplier DECIMAL(4,2) NOT NULL DEFAULT 1.0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE data_tier_category IS 'Top-level categories: Weather, Satellite, Derived';

-- Tier within category (Tier 1, 2, 3 within Weather/Satellite/Derived)
CREATE TABLE data_tier (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    data_tier_category_id UUID NOT NULL REFERENCES data_tier_category(id),
    tier_level INT NOT NULL,
    tier_name VARCHAR(50) NOT NULL,
    data_tier_multiplier DECIMAL(4,2) NOT NULL DEFAULT 1.0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_tier_level CHECK (tier_level > 0),
    CONSTRAINT positive_multiplier CHECK (data_tier_multiplier > 0),
    CONSTRAINT unique_tier_per_category UNIQUE (data_tier_category_id, tier_level)
);

CREATE INDEX idx_data_tier_category ON data_tier(data_tier_category_id);

COMMENT ON TABLE data_tier IS 'Tiers within each category (e.g., Weather Tier 1, Weather Tier 2)';

-- Main data source management (specific parameters)
CREATE TABLE data_source (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Source identification
    data_source data_source_type NOT NULL,
    
    -- Parameter definition
    parameter_name VARCHAR(100) NOT NULL,
    parameter_type parameter_type NOT NULL DEFAULT 'numeric',
    unit VARCHAR(20),
    support_component BOOLEAN NOT NULL DEFAULT false,
    
    -- Display names
    display_name_vi VARCHAR(100),
    description_vi TEXT,
    
    -- Validation ranges
    min_value DECIMAL(10,4),
    max_value DECIMAL(10,4),
    
    -- Technical specifications
    update_frequency VARCHAR(50),
    spatial_resolution VARCHAR(50),
    accuracy_rating DECIMAL(3,2),
    
    -- BASE COST per policy per month
    base_cost BIGINT NOT NULL DEFAULT 0.0,
    
    -- Tier assignment
    data_tier_id UUID NOT NULL REFERENCES data_tier(id),
    
    -- API integration
    data_provider VARCHAR(200),
    api_endpoint VARCHAR(500),
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_base_cost CHECK (base_cost >= 0),
    CONSTRAINT valid_accuracy CHECK (accuracy_rating >= 0 AND accuracy_rating <= 1)
);

CREATE INDEX idx_data_source_tier ON data_source(data_tier_id);
CREATE INDEX idx_data_source_parameter ON data_source(parameter_name);
CREATE INDEX idx_data_source_type ON data_source(data_source);
CREATE INDEX idx_data_source_active ON data_source(is_active) WHERE is_active = true;

COMMENT ON TABLE data_source IS 'Specific data parameters (rainfall, NDVI, etc) within tiers';
COMMENT ON COLUMN data_source.base_cost IS 'Base cost per policy per month in USD';

-- ============================================================================
-- FARM MANAGEMENT
-- ============================================================================

CREATE TABLE farm (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id VARCHAR(100) NOT NULL,
    
    -- Identification
    farm_name VARCHAR(200),
    farm_code VARCHAR(50) UNIQUE,
    
    -- Location
    boundary GEOMETRY(Polygon, 4326),
    center_location GEOGRAPHY(Point, 4326),
    agro_polygon_id VARCHAR(50),
    area_sqm DECIMAL(12,2) NOT NULL,
    
    -- Address
    province VARCHAR(100),
    district VARCHAR(100),
    commune VARCHAR(100),
    address TEXT,
    
    -- Crop information
    crop_type VARCHAR(50) NOT NULL,
    planting_date INT,
    expected_harvest_date INT,
    
    -- Verification
    crop_type_verified BOOLEAN DEFAULT false,
    crop_type_verified_at INT,
    crop_type_verified_by VARCHAR(50),
    crop_type_confidence DECIMAL(3,2),
    
    -- Land ownership
    land_certificate_number VARCHAR(100),
    land_certificate_url VARCHAR(500),
    land_ownership_verified BOOLEAN DEFAULT false,
    land_ownership_verified_at INT,
    
    -- Farm features
    has_irrigation BOOLEAN DEFAULT false,
    irrigation_type VARCHAR(50),
    soil_type VARCHAR(100),
    
    -- Status
    status farm_status DEFAULT 'active',
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_area CHECK (area_sqm > 0)
);

CREATE INDEX idx_farm_owner ON farm(owner_id);
CREATE INDEX idx_farm_crop_type ON farm(crop_type);
CREATE INDEX idx_farm_status ON farm(status);
CREATE INDEX idx_farm_location ON farm USING GIST(center_location);
CREATE INDEX idx_farm_boundary ON farm USING GIST(boundary);
CREATE INDEX idx_farm_planting_date ON farm(planting_date);
CREATE INDEX idx_farm_agro_polygon ON farm(agro_polygon_id);

COMMENT ON COLUMN farm.owner_id IS 'External user service reference';

CREATE TABLE farm_photo (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    farm_id UUID NOT NULL REFERENCES farm(id) ON DELETE CASCADE,
    photo_url VARCHAR(500) NOT NULL,
    photo_type photo_type DEFAULT 'other',
    taken_at INT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_farm_photo_farm ON farm_photo(farm_id);
CREATE INDEX idx_farm_photo_type ON farm_photo(photo_type);

-- ============================================================================
-- BASE POLICY (TEMPLATE/PRODUCT)
-- ============================================================================

CREATE TABLE base_policy (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    insurance_provider_id VARCHAR(100) NOT NULL,
    
    -- Product identification
    product_name VARCHAR(200) NOT NULL,
    product_code VARCHAR(50) UNIQUE,
    product_description TEXT,
    
    -- Coverage parameters
    crop_type VARCHAR(50) NOT NULL,
    coverage_currency VARCHAR(3) DEFAULT 'VND',
    coverage_duration_days INT NOT NULL,
    
    -- Premium formula parameters
    fix_premium_amount INT NOT NULL,
    is_per_hectare BOOLEAN NOT NULL DEFAULT false,
    premium_base_rate DECIMAL(10,4) NOT NULL,
    max_premium_payment_prolong BIGINT,

    -- Payout formula parameters
    fix_payout_amount INT NOT NULL, 
    is_payout_per_hectare BOOLEAN NOT NULL DEFAULT false,
    over_threshold_multiplier DECIMAL(10,4) NOT NULL,
    payout_base_rate DECIMAL(10,4) NOT NULL,
    payout_cap INT,

    -- Cancel config
    cancel_premium_rate DECIMAL(4,2) NOT NULL DEFAULT 1.0,
    
    -- Enrollment date
    enrollment_start_day INT,
    enrollment_end_day INT,
    
    -- Lifecycle
    auto_renewal BOOLEAN DEFAULT false,
    renewal_discount_rate DECIMAL(3,2),
    base_policy_invalid_date INT,
    insurance_valid_from_day INT,
    insurance_valid_to_day INT,
    -- Status
    status base_policy_status DEFAULT 'draft',
    
    -- Documents
    template_document_url VARCHAR(500),
    document_validation_status validation_status DEFAULT 'pending',
    document_validation_score DECIMAL(3,2),
    document_tags JSONB,
    important_additional_information TEXT,
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by VARCHAR(100),
    
    CONSTRAINT positive_premium_rate CHECK (premium_base_rate >= 0),
    CONSTRAINT positive_duration CHECK (coverage_duration_days > 0)
);

CREATE INDEX idx_base_policy_provider ON base_policy(insurance_provider_id);
CREATE INDEX idx_base_policy_status ON base_policy(status);
CREATE INDEX idx_base_policy_crop ON base_policy(crop_type);

COMMENT ON TABLE base_policy IS 'Policy templates - data_tier removed, can use multiple data sources from different tiers';

-- Base policy trigger (ONE trigger group per policy)
CREATE TABLE base_policy_trigger (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    base_policy_id UUID NOT NULL REFERENCES base_policy(id) ON DELETE CASCADE,
    
    -- Logic operator for combining conditions
    logical_operator logical_operator NOT NULL DEFAULT 'AND',
    
    -- Time constraints
    growth_stage VARCHAR(50),
    monitor_interval INT DEFAULT 1,
    monitor_frequency_unit monitor_frequency NOT NULL DEFAULT 'day',
    blackout_periods JSONB,
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT one_trigger_per_base_policy UNIQUE (base_policy_id)
);

CREATE INDEX idx_base_policy_trigger_policy ON base_policy_trigger(base_policy_id);

COMMENT ON TABLE base_policy_trigger IS 'ONE trigger group per base policy, contains multiple conditions';

-- Base policy trigger conditions (multiple conditions per trigger)
-- MERGED: Now includes data usage cost tracking (previously in base_policy_data_usage)
CREATE TABLE base_policy_trigger_condition (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    base_policy_trigger_id UUID NOT NULL REFERENCES base_policy_trigger(id) ON DELETE CASCADE,
    
    -- Data source (THIS is where data sources are selected)
    data_source_id UUID NOT NULL REFERENCES data_source(id),
    
    -- Threshold configuration
    threshold_operator threshold_operator NOT NULL,
    threshold_value DECIMAL(10,4) NOT NULL,
    early_warning_threshold DECIMAL(10,4),
    
    -- Aggregation
    aggregation_function aggregation_function NOT NULL DEFAULT 'avg',
    aggregation_window_days INT NOT NULL,
    consecutive_required BOOLEAN DEFAULT false,
    
    -- Data component
    include_component BOOLEAN NOT NULL DEFAULT false,
    -- Baseline
    baseline_window_days INT,
    baseline_function aggregation_function DEFAULT 'avg',
    
    -- Validation
    validation_window_days INT DEFAULT 7,
    
    -- Order
    condition_order INT DEFAULT 0,
    
    -- Data usage cost tracking (merged from base_policy_data_usage)
    base_cost BIGINT NOT NULL DEFAULT 0.0,
    category_multiplier DECIMAL(4,2) NOT NULL DEFAULT 1.0,
    tier_multiplier DECIMAL(4,2) NOT NULL DEFAULT 1.0,
    calculated_cost DECIMAL(14,4) NOT NULL DEFAULT 0.0,
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_window CHECK (aggregation_window_days > 0),
    CONSTRAINT positive_costs CHECK (calculated_cost >= 0 AND base_cost >= 0)
);


CREATE INDEX idx_base_trigger_condition_trigger ON base_policy_trigger_condition(base_policy_trigger_id);
CREATE INDEX idx_base_trigger_condition_data_source ON base_policy_trigger_condition(data_source_id);
CREATE INDEX idx_base_trigger_condition_order ON base_policy_trigger_condition(base_policy_trigger_id, condition_order);

COMMENT ON TABLE base_policy_trigger_condition IS 'Multiple conditions in a trigger group, each condition references a data_source and includes cost tracking';
COMMENT ON COLUMN base_policy_trigger_condition.calculated_cost IS 'base_cost × category_multiplier × tier_multiplier per month';

-- Document validation for base policy
CREATE TABLE base_policy_document_validation (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    base_policy_id UUID NOT NULL REFERENCES base_policy(id),
    
    validation_timestamp INT NOT NULL,
    validation_status validation_status DEFAULT 'pending',
    overall_score DECIMAL(3,2),
    
    total_checks INT DEFAULT 0,
    passed_checks INT DEFAULT 0,
    failed_checks INT DEFAULT 0,
    warning_count INT DEFAULT 0,
    
    mismatches JSONB,
    warnings JSONB,
    recommendations JSONB,
    extracted_parameters JSONB,
    
    validated_by VARCHAR(100),
    validation_notes TEXT,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_base_doc_validation_policy ON base_policy_document_validation(base_policy_id);
CREATE INDEX idx_base_doc_validation_status ON base_policy_document_validation(validation_status);

-- ============================================================================
-- REGISTERED POLICY (ACTUAL POLICY INSTANCES)
-- ============================================================================

CREATE TABLE registered_policy (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_number VARCHAR(50) NOT NULL UNIQUE,
    
    -- References
    base_policy_id UUID NOT NULL REFERENCES base_policy(id),
    insurance_provider_id VARCHAR(100) NOT NULL,
    farm_id UUID NOT NULL REFERENCES farm(id),
    farmer_id VARCHAR(100) NOT NULL,
    
    -- Coverage
    coverage_amount DECIMAL(12,2) NOT NULL,
    -- signed policy Lifecycle
    coverage_start_date INT NOT NULL,
    coverage_end_date INT NOT NULL,
    planting_date INT,
    
    -- Farmer premium
    area_multiplier DECIMAL(8,2) NOT NULL,
    total_farmer_premium DECIMAL(10,2) NOT NULL,
    premium_paid_by_farmer BOOLEAN DEFAULT false,
    premium_paid_at INT,
    
    -- Agrisa revenue (snapshot from base_policy at registration time)
    data_complexity_score INT NOT NULL,
    monthly_data_cost DECIMAL(10,2) NOT NULL,
    total_data_cost DECIMAL(10,2) NOT NULL,
    
    -- Status
    status policy_status DEFAULT 'draft',
    underwriting_status underwriting_status DEFAULT 'pending',
    
    -- Documents
    signed_policy_document_url VARCHAR(500),
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    registered_by VARCHAR(100),
    
    CONSTRAINT positive_coverage CHECK (coverage_amount > 0),
    CONSTRAINT positive_premium CHECK (total_farmer_premium >= 0),
    CONSTRAINT valid_dates CHECK (coverage_end_date > coverage_start_date)
);

CREATE INDEX idx_registered_policy_base ON registered_policy(base_policy_id);
CREATE INDEX idx_registered_policy_farm ON registered_policy(farm_id);
CREATE INDEX idx_registered_policy_farmer ON registered_policy(farmer_id);
CREATE INDEX idx_registered_policy_provider ON registered_policy(insurance_provider_id);
CREATE INDEX idx_registered_policy_status ON registered_policy(status);
CREATE INDEX idx_registered_policy_dates ON registered_policy(coverage_start_date, coverage_end_date);
CREATE INDEX idx_registered_policy_number ON registered_policy(policy_number);

COMMENT ON TABLE registered_policy IS 'Policy instances - data_complexity_score and costs are snapshots from base_policy';
COMMENT ON COLUMN registered_policy.data_complexity_score IS 'Snapshot from base_policy at registration time';
COMMENT ON COLUMN registered_policy.monthly_data_cost IS 'Snapshot from base_policy at registration time';
COMMENT ON COLUMN registered_policy.total_data_cost IS 'monthly_data_cost × coverage_months';

CREATE TABLE registered_policy_underwriting (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    registered_policy_id UUID NOT NULL REFERENCES registered_policy(id) ON DELETE CASCADE,
    
    validation_timestamp INT NOT NULL,
    underwriting_status underwriting_status DEFAULT 'pending',
    
    recommendations JSONB,
    
    reason TEXT,
    reason_evidence JSONB,

    validated_by VARCHAR(100),
    validation_notes TEXT,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_policy_underwriting ON registered_policy_underwriting(registered_policy_id);
CREATE INDEX idx_policy_underwriting_status ON registered_policy_underwriting(underwriting_status);

CREATE TABLE registered_policy_risk_analysis (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    registered_policy_id UUID NOT NULL REFERENCES registered_policy(id) ON DELETE CASCADE,
    
    -- Analysis Status & Type
    analysis_status validation_status DEFAULT 'pending',
    analysis_type risk_analysis_type NOT NULL,
    analysis_source VARCHAR(100),
    
    -- Analysis Timestamp
    analysis_timestamp INT NOT NULL,
    
    -- High-Level Results
    overall_risk_score DECIMAL(5,4),
    overall_risk_level risk_level, -- 'low', 'medium', 'high', 'critical'
    
    -- Detailed Findings (JSONB)
    identified_risks JSONB, -- Array of objects: [{"risk_code": "BOUNDARY_MISMATCH", "description": "Farm boundary overlaps with existing policy", "score": 0.85}, ...]
    recommendations JSONB, -- Array of strings: ["MANUAL_REVIEW_REQUIRED", "REJECT_APPLICATION"]
    raw_output JSONB, -- Full JSON response from the analysis engine (e.g., AI model)
    
    -- Metadata
    analysis_notes TEXT, -- Human-readable summary or notes
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_risk_analysis_policy ON registered_policy_risk_analysis(registered_policy_id);
CREATE INDEX idx_risk_analysis_status ON registered_policy_risk_analysis(analysis_status);
CREATE INDEX idx_risk_analysis_level ON registered_policy_risk_analysis(overall_risk_level);
CREATE INDEX idx_risk_analysis_type ON registered_policy_risk_analysis(analysis_type);

-- Comments
COMMENT ON TABLE registered_policy_risk_analysis IS 'Stores AI/document-based risk analysis results for a specific policy application.';
COMMENT ON COLUMN registered_policy_risk_analysis.registered_policy_id IS 'Link to the specific policy application being analyzed.';
COMMENT ON COLUMN registered_policy_risk_analysis.analysis_status IS 'Overall status of the analysis job (pending, passed, failed, etc).';
COMMENT ON COLUMN registered_policy_risk_analysis.analysis_type IS 'The method used for the risk analysis (AI, document, etc).';
COMMENT ON COLUMN registered_policy_risk_analysis.overall_risk_score IS 'Normalized risk score (e.g., 0.0 = low, 1.0 = high).';
COMMENT ON COLUMN registered_policy_risk_analysis.identified_risks IS 'JSON array of specific risk factors identified.';
COMMENT ON COLUMN registered_policy_risk_analysis.recommendations IS 'JSON array of suggested actions for underwriting (e.g., MANUAL_REVIEW).';
COMMENT ON COLUMN registered_policy_risk_analysis.raw_output IS 'Full raw JSON response from the analysis engine for debugging.';

CREATE TABLE cancel_request (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    registered_policy_id UUID NOT NULL REFERENCES registered_policy(id) ON DELETE CASCADE,
    
    -- Request details
    cancel_request_type cancel_request_type NOT NULL,
    reason TEXT NOT NULL,
    evidence JSONB,
    
    -- Request status and processing
    status cancel_request_status NOT NULL DEFAULT 'denied',
    requested_by VARCHAR(100) NOT NULL, -- farmer_id or insurance_provider_id
    requested_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Processing details
    reviewed_by VARCHAR(100), -- admin/underwriter who processed
    reviewed_at TIMESTAMP,
    review_notes TEXT,
    -- 
    compensate_amount INT NOT NULL DEFAULT 0,
    paid boolean,
    paid_at TIMESTAMP, 
    during_notice_period boolean,
    
    -- Audit trail
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
);

-- Cancel request indexes for performance
CREATE INDEX idx_cancel_request_policy ON cancel_request(registered_policy_id);
CREATE INDEX idx_cancel_request_status ON cancel_request(status);
CREATE INDEX idx_cancel_request_type ON cancel_request(cancel_request_type);
CREATE INDEX idx_cancel_request_requested_by ON cancel_request(requested_by);
CREATE INDEX idx_cancel_request_requested_at ON cancel_request(requested_at DESC);
CREATE INDEX idx_cancel_request_reviewed_by ON cancel_request(reviewed_by) WHERE reviewed_by IS NOT NULL;
CREATE INDEX idx_cancel_request_pending ON cancel_request(requested_at) WHERE status = 'denied' AND reviewed_by IS NULL;

-- ============================================================================
-- CLAIMS & PAYOUTS
-- ============================================================================

CREATE TABLE claim (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    claim_number VARCHAR(50) NOT NULL UNIQUE,
    
    registered_policy_id UUID NOT NULL REFERENCES registered_policy(id),
    base_policy_id UUID NOT NULL REFERENCES base_policy(id),
    farm_id UUID NOT NULL REFERENCES farm(id),
    base_policy_trigger_id UUID NOT NULL REFERENCES base_policy_trigger(id),
    
    trigger_timestamp INT NOT NULL,
    over_threshold_value DECIMAL(10,4),
    calculated_fix_payout DECIMAL(12,2),
    calculated_threshold_payout DECIMAL(12,2),
    claim_amount DECIMAL(12,2) NOT NULL,
    
    status claim_status DEFAULT 'generated',
    auto_generated BOOLEAN DEFAULT true,
    
    partner_review_timestamp INT,
    partner_decision VARCHAR(20),
    partner_notes TEXT,
    reviewed_by VARCHAR(100),
    
    auto_approval_deadline INT,
    auto_approved BOOLEAN DEFAULT false,
    
    evidence_summary JSONB,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_claim_amount CHECK (claim_amount > 0)
);

CREATE INDEX idx_claim_registered_policy ON claim(registered_policy_id);
CREATE INDEX idx_claim_base_policy ON claim(base_policy_id);
CREATE INDEX idx_claim_farm ON claim(farm_id);
CREATE INDEX idx_claim_status ON claim(status);
CREATE INDEX idx_claim_trigger_timestamp ON claim(trigger_timestamp);
CREATE INDEX idx_claim_number ON claim(claim_number);

CREATE TABLE claim_rejection (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    claim_id UUID NOT NULL REFERENCES claim(id),
    
    validation_timestamp INT NOT NULL,
    claim_rejection_type claim_rejection_type DEFAULT 'claim_data_incorrect',
    
    reason TEXT,
    reason_evidence JSONB,

    validated_by VARCHAR(100),
    validation_notes TEXT,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_claim_rejection_claim ON claim_rejection(claim_id);
CREATE INDEX idx_claim_rejection_type ON claim_rejection(claim_rejection_type);

CREATE TABLE payout (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    claim_id UUID NOT NULL REFERENCES claim(id),
    registered_policy_id UUID NOT NULL REFERENCES registered_policy(id),
    farm_id UUID NOT NULL REFERENCES farm(id),
    farmer_id VARCHAR(100) NOT NULL,
    
    payout_amount DECIMAL(12,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'VND',
    
    status payout_status DEFAULT 'pending',
    initiated_at INT,
    completed_at INT,
    
    farmer_confirmed BOOLEAN DEFAULT false,
    farmer_confirmation_timestamp INT,
    farmer_rating INT,
    farmer_feedback TEXT,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_payout CHECK (payout_amount > 0),
    CONSTRAINT valid_rating CHECK (farmer_rating IS NULL OR (farmer_rating >= 1 AND farmer_rating <= 5))
);

CREATE INDEX idx_payout_claim ON payout(claim_id);
CREATE INDEX idx_payout_registered_policy ON payout(registered_policy_id);
CREATE INDEX idx_payout_farmer ON payout(farmer_id);
CREATE INDEX idx_payout_status ON payout(status);

-- ============================================================================
-- MONITORING DATA
-- ============================================================================

CREATE TABLE farm_monitoring_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    farm_id UUID NOT NULL REFERENCES farm(id),
    data_source_id UUID NOT NULL REFERENCES data_source(id),
    
    parameter_name VARCHAR(100) NOT NULL,
    measured_value DECIMAL(10,4) NOT NULL,
    unit VARCHAR(20),
    measurement_timestamp INT NOT NULL,
    component_data JSONB,
    
    data_quality data_quality DEFAULT 'good',
    confidence_score DECIMAL(3,2),
    
    measurement_source VARCHAR(200),
    distance_from_farm_meters DECIMAL(8,2),
    cloud_cover_percentage DECIMAL(5,2),
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_farm_monitoring_farm_time ON farm_monitoring_data(farm_id, measurement_timestamp);
CREATE INDEX idx_farm_monitoring_data_source ON farm_monitoring_data(data_source_id);
CREATE INDEX idx_farm_monitoring_parameter ON farm_monitoring_data(parameter_name);

-- ============================================================================
-- BILLING & INVOICING
-- ============================================================================

CREATE TABLE partner_invoice (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    insurance_provider_id VARCHAR(100) NOT NULL,
    
    invoice_month INT NOT NULL,
    invoice_number VARCHAR(50) NOT NULL UNIQUE,
    
    active_policies_count INT DEFAULT 0,
    total_data_complexity_fee DECIMAL(12,2) DEFAULT 0,
    
    subtotal DECIMAL(12,2) NOT NULL,
    tax DECIMAL(12,2) DEFAULT 0,
    total_due DECIMAL(12,2) NOT NULL,
    
    payment_status payment_status DEFAULT 'pending',
    due_date INT NOT NULL,
    paid_date INT,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_amounts CHECK (total_due >= 0)
);

CREATE INDEX idx_invoice_provider ON partner_invoice(insurance_provider_id);
CREATE INDEX idx_invoice_month ON partner_invoice(invoice_month);
CREATE INDEX idx_invoice_status ON partner_invoice(payment_status);
CREATE INDEX idx_invoice_number ON partner_invoice(invoice_number);

CREATE TABLE invoice_line_item (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_id UUID NOT NULL REFERENCES partner_invoice(id) ON DELETE CASCADE,
    
    item_type VARCHAR(50) NOT NULL,
    base_policy_id UUID REFERENCES base_policy(id),
    registered_policy_id UUID REFERENCES registered_policy(id),
    
    description TEXT,
    quantity INT DEFAULT 1,
    unit_cost DECIMAL(10,4) NOT NULL,
    total_cost DECIMAL(10,2) NOT NULL,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_quantity CHECK (quantity > 0),
    CONSTRAINT positive_cost CHECK (total_cost >= 0)
);

CREATE INDEX idx_line_item_invoice ON invoice_line_item(invoice_id);
CREATE INDEX idx_line_item_base_policy ON invoice_line_item(base_policy_id);
CREATE INDEX idx_line_item_registered_policy ON invoice_line_item(registered_policy_id);

-- ============================================================================
-- ANALYTICS & LOGGING
-- ============================================================================

CREATE TABLE trigger_evaluation_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    registered_policy_id UUID NOT NULL REFERENCES registered_policy(id),
    base_policy_id UUID NOT NULL REFERENCES base_policy(id),
    farm_id UUID NOT NULL REFERENCES farm(id),
    base_policy_trigger_id UUID NOT NULL REFERENCES base_policy_trigger(id),
    
    evaluation_timestamp INT NOT NULL,
    evaluation_result BOOLEAN NOT NULL,
    
    conditions_evaluated INT DEFAULT 0,
    conditions_met INT DEFAULT 0,
    condition_details JSONB,
    
    claim_generated BOOLEAN DEFAULT false,
    claim_id UUID REFERENCES claim(id),
    
    evaluation_duration_ms INT,
    data_sources_queried INT,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_eval_log_registered_policy_time ON trigger_evaluation_log(registered_policy_id, evaluation_timestamp);
CREATE INDEX idx_eval_log_base_policy ON trigger_evaluation_log(base_policy_id);
CREATE INDEX idx_eval_log_trigger ON trigger_evaluation_log(base_policy_trigger_id);
CREATE INDEX idx_eval_log_result ON trigger_evaluation_log(evaluation_result);

-- ============================================================================
-- WORKER
-- ============================================================================

-- Worker Pool State Table
CREATE TABLE IF NOT EXISTS worker_pool_state (
    policy_id UUID PRIMARY KEY REFERENCES registered_policy(id) ON DELETE CASCADE,
    pool_name VARCHAR(255) NOT NULL UNIQUE,
    queue_name_base VARCHAR(255) NOT NULL,
    num_workers INT NOT NULL CHECK (num_workers > 0),
    job_timeout INTERVAL NOT NULL,
    pool_status VARCHAR(50) NOT NULL CHECK (pool_status IN ('created', 'active', 'stopped', 'archived')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    stopped_at TIMESTAMP,
    last_job_at TIMESTAMP,
    metadata JSONB DEFAULT '{}'::jsonb,

    CONSTRAINT pool_name_format CHECK (pool_name ~ '^policy-[a-f0-9-]+-pool$')
);

-- Worker Scheduler State Table
CREATE TABLE IF NOT EXISTS worker_scheduler_state (
    policy_id UUID PRIMARY KEY REFERENCES registered_policy(id) ON DELETE CASCADE,
    scheduler_name VARCHAR(255) NOT NULL UNIQUE,
    monitor_interval INTERVAL NOT NULL,
    monitor_frequency_unit VARCHAR(20) NOT NULL CHECK (monitor_frequency_unit IN ('hour', 'day', 'week', 'month')),
    scheduler_status VARCHAR(50) NOT NULL CHECK (scheduler_status IN ('created', 'active', 'stopped', 'archived')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    stopped_at TIMESTAMP,
    last_run_at TIMESTAMP,
    next_run_at TIMESTAMP,
    run_count BIGINT DEFAULT 0,
    metadata JSONB DEFAULT '{}'::jsonb,

    CONSTRAINT scheduler_name_format CHECK (scheduler_name ~ '^policy-[a-f0-9-]+-scheduler$')
);

-- Worker Job Execution History Table
CREATE TABLE IF NOT EXISTS worker_job_execution (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id UUID NOT NULL REFERENCES registered_policy(id) ON DELETE CASCADE,
    job_id VARCHAR(255) NOT NULL,
    job_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'retrying')),
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    result_summary JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT job_execution_times CHECK (completed_at IS NULL OR completed_at >= started_at)
);

-- Indexes for Performance
CREATE INDEX IF NOT EXISTS idx_worker_pool_status ON worker_pool_state(pool_status);
CREATE INDEX IF NOT EXISTS idx_worker_pool_last_job ON worker_pool_state(last_job_at);
CREATE INDEX IF NOT EXISTS idx_worker_scheduler_status ON worker_scheduler_state(scheduler_status);
CREATE INDEX IF NOT EXISTS idx_worker_scheduler_next_run ON worker_scheduler_state(next_run_at);
CREATE INDEX IF NOT EXISTS idx_worker_job_policy_id ON worker_job_execution(policy_id);
CREATE INDEX IF NOT EXISTS idx_worker_job_status ON worker_job_execution(status);
CREATE INDEX IF NOT EXISTS idx_worker_job_created_at ON worker_job_execution(created_at DESC);

-- Comments for Documentation
COMMENT ON TABLE worker_pool_state IS 'Persistence state for worker pools tied to registered policies';
COMMENT ON TABLE worker_scheduler_state IS 'Persistence state for schedulers tied to registered policies';
COMMENT ON TABLE worker_job_execution IS 'Execution history and status of worker jobs';

-- ============================================================================
-- SAMPLE DATA
-- ============================================================================

INSERT INTO data_tier_category (category_name, category_description, category_cost_multiplier) VALUES
    ('Weather', 'Basic weather data from meteorological stations', 1.0),
    ('Satellite', 'Satellite imagery and derived indices', 1.5),
    ('Derived', 'Advanced calculated indices and analytics', 2.5);

DO $$
DECLARE
    weather_cat_id UUID;
    satellite_cat_id UUID;
    derived_cat_id UUID;
BEGIN
    SELECT id INTO weather_cat_id FROM data_tier_category WHERE category_name = 'Weather';
    SELECT id INTO satellite_cat_id FROM data_tier_category WHERE category_name = 'Satellite';
    SELECT id INTO derived_cat_id FROM data_tier_category WHERE category_name = 'Derived';
    
    INSERT INTO data_tier (data_tier_category_id, tier_level, tier_name, data_tier_multiplier) VALUES
        (weather_cat_id, 1, 'Weather Tier 1', 1.0),
        (weather_cat_id, 2, 'Weather Tier 2', 1.2),
        (weather_cat_id, 3, 'Weather Tier 3', 1.5),
        (satellite_cat_id, 1, 'Satellite Tier 1', 1.0),
        (satellite_cat_id, 2, 'Satellite Tier 2', 1.3),
        (satellite_cat_id, 3, 'Satellite Tier 3', 1.6),
        (derived_cat_id, 1, 'Derived Tier 1', 1.0),
        (derived_cat_id, 2, 'Derived Tier 2', 1.4);
END
$$ LANGUAGE plpgsql;


