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
CREATE TYPE base_policy_status AS ENUM ('draft', 'active', 'archived');
CREATE TYPE policy_status AS ENUM ('draft', 'pending_review', 'active', 'expired', 'cancelled');
CREATE TYPE underwriting_status AS ENUM ('pending', 'approved', 'rejected');
CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'overdue', 'cancelled', 'refunded');
CREATE TYPE validation_status AS ENUM ('pending', 'passed', 'failed', 'warning');
CREATE TYPE threshold_operator AS ENUM ('<', '>', '<=', '>=', '==', '!=', 'change_gt', 'change_lt');
CREATE TYPE aggregation_function AS ENUM ('sum', 'avg', 'min', 'max', 'change');
CREATE TYPE logical_operator AS ENUM ('AND', 'OR');
CREATE TYPE claim_status AS ENUM ('generated', 'pending_partner_review', 'approved', 'rejected', 'paid');
CREATE TYPE payout_status AS ENUM ('pending', 'processing', 'completed', 'failed');
CREATE TYPE data_quality AS ENUM ('good', 'acceptable', 'poor');
CREATE TYPE farm_status AS ENUM ('active', 'inactive', 'archived');
CREATE TYPE photo_type AS ENUM ('crop', 'boundary', 'land_certificate', 'other');
CREATE TYPE monitor_frequency AS ENUM ('hour', 'day', 'week', 'month', 'year')

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
    base_cost DECIMAL(8,4) NOT NULL DEFAULT 0.0,
    
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

    -- Payout formula parameters
    fix_payout_amount INT NOT NULL, 
    is_payout_per_hectare BOOLEAN NOT NULL DEFAULT false,
    over_threshold_multiplier DECIMAL(10,4) NOT NULL,
    payout_base_rate DECIMAL(10,4) NOT NULL,
    
    -- Data complexity (calculated from base_policy_data_usage)
    data_complexity_score INT DEFAULT 0,
    monthly_data_cost DECIMAL(10,2) DEFAULT 0,
    
    -- Status
    status base_policy_status DEFAULT 'draft',
    
    -- Documents
    template_document_url VARCHAR(500),
    document_validation_status validation_status DEFAULT 'pending',
    document_validation_score DECIMAL(3,2),
    important_additional_information JSONB,
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by VARCHAR(100),
    
    CONSTRAINT positive_premium_rate CHECK (premium_base_rate >= 0),
    CONSTRAINT positive_duration CHECK (coverage_duration_days > 0),
    CONSTRAINT positive_complexity CHECK (data_complexity_score >= 0),
    CONSTRAINT positive_data_cost CHECK (monthly_data_cost >= 0)
);

CREATE INDEX idx_base_policy_provider ON base_policy(insurance_provider_id);
CREATE INDEX idx_base_policy_status ON base_policy(status);
CREATE INDEX idx_base_policy_crop ON base_policy(crop_type);
CREATE INDEX idx_base_policy_complexity ON base_policy(data_complexity_score);

COMMENT ON TABLE base_policy IS 'Policy templates - data_tier removed, can use multiple data sources from different tiers';
COMMENT ON COLUMN base_policy.data_complexity_score IS 'Number of unique data sources used (calculated)';
COMMENT ON COLUMN base_policy.monthly_data_cost IS 'Total monthly cost for all data sources (calculated)';

-- Base policy trigger (ONE trigger group per policy)
CREATE TABLE base_policy_trigger (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    base_policy_id UUID NOT NULL REFERENCES base_policy(id) ON DELETE CASCADE,
    
    -- Logic operator for combining conditions
    logical_operator logical_operator NOT NULL DEFAULT 'AND',
    
    -- Time constraints
    valid_from_day INT,
    valid_to_day INT,
    growth_stage VARCHAR(50),
    monitor_frequency_value INT DEFAULT 1,
    monitor_frequency_unit monitor_frequency NOT NULL DEFAULT 'day',
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT one_trigger_per_base_policy UNIQUE (base_policy_id),
    CONSTRAINT valid_payout_percentage CHECK (payout_percentage >= 0 AND payout_percentage <= 100),
    CONSTRAINT valid_day_range CHECK (valid_to_day IS NULL OR valid_from_day IS NULL OR valid_to_day >= valid_from_day)
);

CREATE INDEX idx_base_policy_trigger_policy ON base_policy_trigger(base_policy_id);

COMMENT ON TABLE base_policy_trigger IS 'ONE trigger group per base policy, contains multiple conditions';

-- Base policy trigger conditions (multiple conditions per trigger)
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
    
    -- Baseline
    baseline_window_days INT,
    baseline_function aggregation_function DEFAULT 'avg',
    
    -- Validation
    validation_window_days INT DEFAULT 7,
    
    -- Order
    condition_order INT DEFAULT 0,
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_window CHECK (aggregation_window_days > 0)
);

CREATE INDEX idx_base_trigger_condition_trigger ON base_policy_trigger_condition(base_policy_trigger_id);
CREATE INDEX idx_base_trigger_condition_data_source ON base_policy_trigger_condition(data_source_id);
CREATE INDEX idx_base_trigger_condition_order ON base_policy_trigger_condition(base_policy_trigger_id, condition_order);

COMMENT ON TABLE base_policy_trigger_condition IS 'Multiple conditions in a trigger group, each condition references a data_source';

-- Track data sources used in base policy (ONE row per data source per policy)
CREATE TABLE base_policy_data_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    base_policy_id UUID NOT NULL REFERENCES base_policy(id) ON DELETE CASCADE,
    data_source_id UUID NOT NULL REFERENCES data_source(id),
    
    -- Cost snapshot at time of selection
    base_cost DECIMAL(8,4) NOT NULL,
    category_multiplier DECIMAL(4,2) NOT NULL,
    tier_multiplier DECIMAL(4,2) NOT NULL,
    calculated_cost DECIMAL(10,4) NOT NULL,
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_costs CHECK (calculated_cost >= 0),
    CONSTRAINT unique_data_source_per_policy UNIQUE (base_policy_id, data_source_id)
);

CREATE INDEX idx_base_policy_data_usage_policy ON base_policy_data_usage(base_policy_id);
CREATE INDEX idx_base_policy_data_usage_data_source ON base_policy_data_usage(data_source_id);

COMMENT ON TABLE base_policy_data_usage IS 'Tracks which data sources are used in base policy (one row per data source)';
COMMENT ON COLUMN base_policy_data_usage.calculated_cost IS 'base_cost × category_multiplier × tier_multiplier per month';

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
    coverage_start_date INT NOT NULL,
    coverage_end_date INT NOT NULL,
    planting_date INT NOT NULL,
    
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
    rejection_reason TEXT,
    
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
END $$;


