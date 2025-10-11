CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";

-- ============================================================================
-- ENUM TYPES
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

-- ============================================================================
-- CORE DATA SOURCE
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
COMMENT ON COLUMN data_tier_category.category_cost_multiplier IS 'Base multiplier for category';

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
COMMENT ON COLUMN data_tier.data_tier_multiplier IS 'Multiplier for this tier level';

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
    
    -- Tier assignment (data_source belongs to a tier)
    data_tier_id UUID NOT NULL REFERENCES data_tier(id),
    data_complexity_score DECIMAL(8,4) NOT NULL DEFAULT 0.0,
    
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
    owner_id VARCHAR(100) NOT NULL, -- External user service reference
    
    -- Identification
    farm_name VARCHAR(200),
    farm_code VARCHAR(50) UNIQUE,
    
    -- Location (PostGIS geography types)
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
    planting_date INT, -- unix timestamp
    expected_harvest_date INT, -- unix timestamp
    
    -- Verification
    crop_type_verified BOOLEAN DEFAULT false,
    crop_type_verified_at INT, -- unix timestamp
    crop_type_verified_by VARCHAR(50),
    crop_type_confidence DECIMAL(3,2),
    
    -- Land ownership
    land_certificate_number VARCHAR(100),
    land_certificate_url VARCHAR(500),
    land_ownership_verified BOOLEAN DEFAULT false,
    land_ownership_verified_at INT, -- unix timestamp
    
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

COMMENT ON TABLE farm IS 'Farm management - owner_id references external user service';
COMMENT ON COLUMN farm.owner_id IS 'External user service reference (user ID)';
COMMENT ON COLUMN farm.planting_date IS 'Unix timestamp - critical for calculating growth stages';

CREATE TABLE farm_photo (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    farm_id UUID NOT NULL REFERENCES farm(id) ON DELETE CASCADE,
    photo_url VARCHAR(500) NOT NULL,
    photo_type photo_type DEFAULT 'other',
    taken_at INT, -- unix timestamp
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_farm_photo_farm ON farm_photo(farm_id);
CREATE INDEX idx_farm_photo_type ON farm_photo(photo_type);

-- ============================================================================
-- BASE POLICY (TEMPLATE/PRODUCT)
-- ============================================================================

CREATE TABLE base_policy (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    insurance_provider_id VARCHAR(100) NOT NULL, -- External provider service reference
    
    -- Importance additional information
    essential_additional_infomation JSONB,
    -- Product identification
    product_name VARCHAR(200) NOT NULL,
    product_code VARCHAR(50) UNIQUE,
    product_description TEXT,
    
    -- Coverage parameters (template)
    crop_type VARCHAR(50) NOT NULL,
    coverage_currency VARCHAR(3) DEFAULT 'VND',
    
    -- Coverage period (days)
    coverage_duration_days INT NOT NULL,
    coverage_start_day_rule VARCHAR(200),
    
    -- Premium formula parameters
    premium_base_rate DECIMAL(10,4) NOT NULL,
    
    -- Data complexity
    data_tier_id UUID NOT NULL REFERENCES data_tier(id),
    data_complexity_score INT DEFAULT 0,
    monthly_data_cost DECIMAL(10,2) DEFAULT 0,
    
    -- Status
    status base_policy_status DEFAULT 'draft',
    
    -- Documents
    template_document_url VARCHAR(500),
    document_validation_status validation_status DEFAULT 'pending',
    document_validation_score DECIMAL(3,2),
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by VARCHAR(100), -- External user service reference
    
    CONSTRAINT positive_premium_rate CHECK (premium_base_rate >= 0),
    CONSTRAINT positive_duration CHECK (coverage_duration_days > 0)
);

CREATE INDEX idx_base_policy_provider ON base_policy(insurance_provider_id);
CREATE INDEX idx_base_policy_status ON base_policy(status);
CREATE INDEX idx_base_policy_tier ON base_policy(data_tier_id);
CREATE INDEX idx_base_policy_crop ON base_policy(crop_type);

COMMENT ON TABLE base_policy IS 'Policy templates/products created by insurance partners';
COMMENT ON COLUMN base_policy.insurance_provider_id IS 'External insurance provider service reference';
COMMENT ON COLUMN base_policy.monthly_data_cost IS 'What partner pays Agrisa per active policy using this template';
COMMENT ON COLUMN base_policy.created_by IS 'External user ID who created this template';

-- Base policy trigger (template trigger)
CREATE TABLE base_policy_trigger (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    base_policy_id UUID NOT NULL REFERENCES base_policy(id) ON DELETE CASCADE,
    
    -- Logic
    logical_operator logical_operator NOT NULL DEFAULT 'AND',
    
    -- Payout
    payout_percentage DECIMAL(5,2) NOT NULL,
    
    -- Time constraints (relative to planting)
    valid_from_day INT,
    valid_to_day INT,
    growth_stage VARCHAR(50),
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT one_trigger_per_base_policy UNIQUE (base_policy_id),
    CONSTRAINT valid_payout_percentage CHECK (payout_percentage >= 0 AND payout_percentage <= 100),
    CONSTRAINT valid_day_range CHECK (valid_to_day IS NULL OR valid_from_day IS NULL OR valid_to_day >= valid_from_day)
);

CREATE INDEX idx_base_policy_trigger_policy ON base_policy_trigger(base_policy_id);

COMMENT ON TABLE base_policy_trigger IS 'Trigger configuration for base policy template';

-- Base policy trigger conditions (template conditions)
CREATE TABLE base_policy_trigger_condition (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    base_policy_trigger_id UUID NOT NULL REFERENCES base_policy_trigger(id) ON DELETE CASCADE,
    
    -- Data source
    data_source_id UUID NOT NULL REFERENCES data_source(id),
    
    -- Threshold configuration
    threshold_operator threshold_operator NOT NULL,
    threshold_value DECIMAL(10,4) NOT NULL,
    
    -- Aggregation
    aggregation_function aggregation_function NOT NULL DEFAULT 'avg',
    aggregation_window_days INT NOT NULL,
    consecutive_required BOOLEAN DEFAULT false,
    
    -- Baseline (for change calculations)
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

-- Track data sources used in base policy (for cost calculation)
CREATE TABLE base_policy_data_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    base_policy_id UUID NOT NULL REFERENCES base_policy(id) ON DELETE CASCADE,
    data_source_id UUID NOT NULL REFERENCES data_source(id),
    
    -- Cost snapshot
    base_cost DECIMAL(8,4) NOT NULL,
    category_multiplier DECIMAL(4,2) NOT NULL,
    tier_multiplier DECIMAL(4,2) NOT NULL,
    calculated_cost DECIMAL(10,4) NOT NULL,
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_costs CHECK (calculated_cost >= 0)
);

CREATE INDEX idx_base_policy_data_usage_policy ON base_policy_data_usage(base_policy_id);
CREATE INDEX idx_base_policy_data_usage_data_source ON base_policy_data_usage(data_source_id);

COMMENT ON TABLE base_policy_data_usage IS 'Data sources used in base policy template';

-- Document validation for base policy
CREATE TABLE base_policy_document_validation (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    base_policy_id UUID NOT NULL REFERENCES base_policy(id),
    
    validation_timestamp INT NOT NULL, -- unix timestamp
    validation_status validation_status DEFAULT 'pending',
    overall_score DECIMAL(3,2),
    
    -- Statistics
    total_checks INT DEFAULT 0,
    passed_checks INT DEFAULT 0,
    failed_checks INT DEFAULT 0,
    warning_count INT DEFAULT 0,
    
    -- Details (JSONB for flexibility)
    mismatches JSONB,
    warnings JSONB,
    recommendations JSONB,
    extracted_parameters JSONB,
    
    -- Review
    validated_by VARCHAR(100), -- External user service reference
    validation_notes TEXT,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_base_doc_validation_policy ON base_policy_document_validation(base_policy_id);
CREATE INDEX idx_base_doc_validation_status ON base_policy_document_validation(validation_status);

COMMENT ON TABLE base_policy_document_validation IS 'NLP validation for base policy template documents';
COMMENT ON COLUMN base_policy_document_validation.validated_by IS 'External user ID who validated';

-- ============================================================================
-- REGISTERED POLICY (ACTUAL POLICY INSTANCES)
-- ============================================================================

CREATE TABLE registered_policy (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_number VARCHAR(50) NOT NULL UNIQUE,
    
    -- References
    base_policy_id UUID NOT NULL REFERENCES base_policy(id),
    insurance_provider_id VARCHAR(100) NOT NULL, -- External provider service reference
    farm_id UUID NOT NULL REFERENCES farm(id),
    farmer_id VARCHAR(100) NOT NULL, -- External user service reference
    
    -- Coverage (specific to this registration)
    coverage_amount DECIMAL(12,2) NOT NULL,
    coverage_start_date INT NOT NULL, -- unix timestamp
    coverage_end_date INT NOT NULL, -- unix timestamp
    planting_date INT NOT NULL, -- unix timestamp (snapshot from farm)
    
    -- Farmer premium (what farmer pays)
    area_multiplier DECIMAL(8,2) NOT NULL,
    total_farmer_premium DECIMAL(10,2) NOT NULL,
    premium_paid_by_farmer BOOLEAN DEFAULT false,
    premium_paid_at INT, -- unix timestamp
    
    -- Agrisa revenue (snapshot from base_policy)
    monthly_data_cost DECIMAL(10,2) NOT NULL,
    total_data_cost DECIMAL(10,2) NOT NULL,
    
    -- Status
    status policy_status DEFAULT 'draft',
    underwriting_status underwriting_status DEFAULT 'pending',
    rejection_reason TEXT,
    
    -- Signed documents
    signed_policy_document_url VARCHAR(500),
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    registered_by VARCHAR(100), -- External user service reference
    
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

COMMENT ON TABLE registered_policy IS 'Actual policy instances tied to specific farms and farmers';
COMMENT ON COLUMN registered_policy.base_policy_id IS 'References the template/product this policy is based on';
COMMENT ON COLUMN registered_policy.insurance_provider_id IS 'External insurance provider service reference';
COMMENT ON COLUMN registered_policy.farmer_id IS 'External user service reference';
COMMENT ON COLUMN registered_policy.total_farmer_premium IS 'What farmer pays to insurance partner';
COMMENT ON COLUMN registered_policy.total_data_cost IS 'What partner pays Agrisa (snapshot from base_policy)';
COMMENT ON COLUMN registered_policy.registered_by IS 'External user ID who registered this policy';

-- ============================================================================
-- CLAIMS & PAYOUTS
-- ============================================================================

CREATE TABLE claim (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    claim_number VARCHAR(50) NOT NULL UNIQUE,
    
    -- Relationships
    registered_policy_id UUID NOT NULL REFERENCES registered_policy(id),
    base_policy_id UUID NOT NULL REFERENCES base_policy(id),
    farm_id UUID NOT NULL REFERENCES farm(id),
    base_policy_trigger_id UUID NOT NULL REFERENCES base_policy_trigger(id),
    
    -- Trigger details
    trigger_timestamp INT NOT NULL, -- unix timestamp
    
    -- Claim amount
    claim_amount DECIMAL(12,2) NOT NULL,
    
    -- Status
    status claim_status DEFAULT 'generated',
    auto_generated BOOLEAN DEFAULT true,
    
    -- Partner review
    partner_review_timestamp INT, -- unix timestamp
    partner_decision VARCHAR(20),
    partner_notes TEXT,
    reviewed_by VARCHAR(100), -- External user service reference
    
    -- Model B: Auto-approval timer
    auto_approval_deadline INT, -- unix timestamp
    auto_approved BOOLEAN DEFAULT false,
    
    -- Evidence
    evidence_summary JSONB,
    
    -- Metadata
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

COMMENT ON TABLE claim IS 'Claims auto-generated by Agrisa for registered policies';
COMMENT ON COLUMN claim.reviewed_by IS 'External user ID who reviewed the claim';

-- Payout tracking
CREATE TABLE payout (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Relationships
    claim_id UUID NOT NULL REFERENCES claim(id),
    registered_policy_id UUID NOT NULL REFERENCES registered_policy(id),
    farm_id UUID NOT NULL REFERENCES farm(id),
    farmer_id VARCHAR(100) NOT NULL, -- External user service reference
    
    -- Amount
    payout_amount DECIMAL(12,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'VND',
    
    -- Status
    status payout_status DEFAULT 'pending',
    initiated_at INT, -- unix timestamp
    completed_at INT, -- unix timestamp
    
    -- Farmer confirmation
    farmer_confirmed BOOLEAN DEFAULT false,
    farmer_confirmation_timestamp INT, -- unix timestamp
    farmer_rating INT,
    farmer_feedback TEXT,
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_payout CHECK (payout_amount > 0),
    CONSTRAINT valid_rating CHECK (farmer_rating IS NULL OR (farmer_rating >= 1 AND farmer_rating <= 5))
);

CREATE INDEX idx_payout_claim ON payout(claim_id);
CREATE INDEX idx_payout_registered_policy ON payout(registered_policy_id);
CREATE INDEX idx_payout_farmer ON payout(farmer_id);
CREATE INDEX idx_payout_status ON payout(status);

COMMENT ON TABLE payout IS 'Partner pays farmer directly, Agrisa tracks status only';
COMMENT ON COLUMN payout.farmer_id IS 'External user service reference';

-- ============================================================================
-- MONITORING DATA (TIME-SERIES)
-- ============================================================================

CREATE TABLE farm_monitoring_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    farm_id UUID NOT NULL REFERENCES farm(id),
    data_source_id UUID NOT NULL REFERENCES data_source(id),
    
    -- Measurement
    parameter_name VARCHAR(100) NOT NULL,
    measured_value DECIMAL(10,4) NOT NULL,
    unit VARCHAR(20),
    measurement_timestamp INT NOT NULL, -- unix timestamp
    
    -- Quality
    data_quality data_quality DEFAULT 'good',
    confidence_score DECIMAL(3,2),
    
    -- Source details
    measurement_source VARCHAR(200),
    distance_from_farm_meters DECIMAL(8,2),
    
    -- For satellite data
    cloud_cover_percentage DECIMAL(5,2),
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_farm_monitoring_farm_time ON farm_monitoring_data(farm_id, measurement_timestamp);
CREATE INDEX idx_farm_monitoring_data_source ON farm_monitoring_data(data_source_id);
CREATE INDEX idx_farm_monitoring_parameter ON farm_monitoring_data(parameter_name);

COMMENT ON TABLE farm_monitoring_data IS 'Time-series monitoring data';
COMMENT ON COLUMN farm_monitoring_data.measurement_timestamp IS 'Unix timestamp for easier calculations';

-- ============================================================================
-- BILLING & INVOICING
-- ============================================================================

CREATE TABLE partner_invoice (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    insurance_provider_id VARCHAR(100) NOT NULL, -- External provider service reference
    
    -- Billing period
    invoice_month INT NOT NULL, -- unix timestamp (first day of month)
    invoice_number VARCHAR(50) NOT NULL UNIQUE,
    
    -- Usage-based (main revenue)
    active_policies_count INT DEFAULT 0,
    total_data_complexity_fee DECIMAL(12,2) DEFAULT 0,
    
    -- Totals
    subtotal DECIMAL(12,2) NOT NULL,
    tax DECIMAL(12,2) DEFAULT 0,
    total_due DECIMAL(12,2) NOT NULL,
    
    -- Payment
    payment_status payment_status DEFAULT 'pending',
    due_date INT NOT NULL, -- unix timestamp
    paid_date INT, -- unix timestamp
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_amounts CHECK (total_due >= 0)
);

CREATE INDEX idx_invoice_provider ON partner_invoice(insurance_provider_id);
CREATE INDEX idx_invoice_month ON partner_invoice(invoice_month);
CREATE INDEX idx_invoice_status ON partner_invoice(payment_status);
CREATE INDEX idx_invoice_number ON partner_invoice(invoice_number);

COMMENT ON TABLE partner_invoice IS 'Monthly invoices to insurance partners for data platform usage';
COMMENT ON COLUMN partner_invoice.insurance_provider_id IS 'External insurance provider service reference';

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
    
    evaluation_timestamp INT NOT NULL, -- unix timestamp
    evaluation_result BOOLEAN NOT NULL,
    
    -- Condition results
    conditions_evaluated INT DEFAULT 0,
    conditions_met INT DEFAULT 0,
    condition_details JSONB,
    
    -- If triggered
    claim_generated BOOLEAN DEFAULT false,
    claim_id UUID REFERENCES claim(id),
    
    -- Performance metrics
    evaluation_duration_ms INT,
    data_sources_queried INT,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_eval_log_registered_policy_time ON trigger_evaluation_log(registered_policy_id, evaluation_timestamp);
CREATE INDEX idx_eval_log_base_policy ON trigger_evaluation_log(base_policy_id);
CREATE INDEX idx_eval_log_trigger ON trigger_evaluation_log(base_policy_trigger_id);
CREATE INDEX idx_eval_log_result ON trigger_evaluation_log(evaluation_result);

COMMENT ON TABLE trigger_evaluation_log IS 'Audit log of all trigger evaluations';

-- ============================================================================
-- SAMPLE DATA
-- ============================================================================

-- Insert sample data tier categories
INSERT INTO data_tier_category (category_name, category_description, category_cost_multiplier) VALUES
    ('Weather', 'Basic weather data from meteorological stations', 1.0),
    ('Satellite', 'Satellite imagery and derived indices', 1.5),
    ('Derived', 'Advanced calculated indices and analytics', 2.5);

-- Insert sample tiers within categories
DO $$
DECLARE
    weather_cat_id UUID;
    satellite_cat_id UUID;
    derived_cat_id UUID;
BEGIN
    SELECT id INTO weather_cat_id FROM data_tier_category WHERE category_name = 'Weather';
    SELECT id INTO satellite_cat_id FROM data_tier_category WHERE category_name = 'Satellite';
    SELECT id INTO derived_cat_id FROM data_tier_category WHERE category_name = 'Derived';
    
    -- Weather tiers
    INSERT INTO data_tier (data_tier_category_id, tier_level, tier_name, data_tier_multiplier) VALUES
        (weather_cat_id, 1, 'Weather Tier 1', 1.0),
        (weather_cat_id, 2, 'Weather Tier 2', 1.2),
        (weather_cat_id, 3, 'Weather Tier 3', 1.5);
    
    -- Satellite tiers
    INSERT INTO data_tier (data_tier_category_id, tier_level, tier_name, data_tier_multiplier) VALUES
        (satellite_cat_id, 1, 'Satellite Tier 1', 1.0),
        (satellite_cat_id, 2, 'Satellite Tier 2', 1.3),
        (satellite_cat_id, 3, 'Satellite Tier 3', 1.6);
    
    -- Derived tiers
    INSERT INTO data_tier (data_tier_category_id, tier_level, tier_name, data_tier_multiplier) VALUES
        (derived_cat_id, 1, 'Derived Tier 1', 1.0),
        (derived_cat_id, 2, 'Derived Tier 2', 1.4);
END $$;

-- ============================================================================
-- FUNCTIONS & TRIGGERS
-- ============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply triggers
CREATE TRIGGER update_data_tier_category_updated_at BEFORE UPDATE ON data_tier_category
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_data_tier_updated_at BEFORE UPDATE ON data_tier
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_data_source_updated_at BEFORE UPDATE ON data_source
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_farm_updated_at BEFORE UPDATE ON farm
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_base_policy_updated_at BEFORE UPDATE ON base_policy
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_base_policy_trigger_updated_at BEFORE UPDATE ON base_policy_trigger
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_registered_policy_updated_at BEFORE UPDATE ON registered_policy
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_claim_updated_at BEFORE UPDATE ON claim
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- VIEWS
-- ============================================================================

-- View: Active registered policies with cost breakdown
CREATE OR REPLACE VIEW v_active_registered_policies AS
SELECT 
    rp.id AS registered_policy_id,
    rp.policy_number,
    bp.product_name AS base_product,
    rp.insurance_provider_id,
    f.farm_name,
    rp.farmer_id,
    rp.coverage_amount,
    rp.total_farmer_premium,
    rp.monthly_data_cost,
    rp.total_data_cost,
    dt.tier_name,
    rp.status,
    rp.coverage_start_date,
    rp.coverage_end_date
FROM registered_policy rp
JOIN base_policy bp ON rp.base_policy_id = bp.id
JOIN farm f ON rp.farm_id = f.id
JOIN data_tier dt ON bp.data_tier_id = dt.id
WHERE rp.status = 'active';

-- View: Base policy catalog
CREATE OR REPLACE VIEW v_base_policy_catalog AS
SELECT 
    bp.id,
    bp.product_name,
    bp.product_code,
    bp.crop_type,
    bp.insurance_provider_id,
    dt.tier_name,
    dtc.category_name AS category,
    bp.monthly_data_cost,
    bp.premium_base_rate,
    bp.status,
    COUNT(rp.id) AS active_policies_count
FROM base_policy bp
JOIN data_tier dt ON bp.data_tier_id = dt.id
JOIN data_tier_category dtc ON dt.data_tier_category_id = dtc.id
LEFT JOIN registered_policy rp ON rp.base_policy_id = bp.id AND rp.status = 'active'
WHERE bp.status = 'active'
GROUP BY bp.id, bp.product_name, bp.product_code, bp.crop_type, 
         bp.insurance_provider_id, dt.tier_name, dtc.category_name, 
         bp.monthly_data_cost, bp.premium_base_rate, bp.status;

-- View: Monthly revenue by provider
CREATE OR REPLACE VIEW v_monthly_revenue_by_provider AS
SELECT 
    rp.insurance_provider_id,
    DATE_TRUNC('month', TO_TIMESTAMP(rp.coverage_start_date)) AS month,
    COUNT(rp.id) AS active_policies,
    SUM(rp.monthly_data_cost) AS total_monthly_revenue
FROM registered_policy rp
WHERE rp.status = 'active'
GROUP BY rp.insurance_provider_id, month
ORDER BY month DESC, total_monthly_revenue DESC;

-- ============================================================================
-- HELPER FUNCTIONS FOR UNIX TIMESTAMP CONVERSION
-- ============================================================================

-- Convert unix timestamp to timestamp
CREATE OR REPLACE FUNCTION unix_to_timestamp(unix_time INT)
RETURNS TIMESTAMP AS $$
BEGIN
    RETURN TO_TIMESTAMP(unix_time);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Convert timestamp to unix timestamp
CREATE OR REPLACE FUNCTION timestamp_to_unix(ts TIMESTAMP)
RETURNS INT AS $$
BEGIN
    RETURN EXTRACT(EPOCH FROM ts)::INT;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Get current unix timestamp
CREATE OR REPLACE FUNCTION current_unix_timestamp()
RETURNS INT AS $$
BEGIN
    RETURN EXTRACT(EPOCH FROM NOW())::INT;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '==============================================';
    RAISE NOTICE 'Agrisa database schema created successfully!';
    RAISE NOTICE '==============================================';
    RAISE NOTICE 'Tables created: %', (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE');
END $$;
