package gemini

import (
	"encoding/json"
	"fmt"
	"policy-service/internal/models"
	"strings"
	"time"
)

const ValidationPromptTemplate = `You are a document validation engine ensuring system data accuracy against official PDF policy documents.

## PRIMARY OBJECTIVE
Validate that the system's database records (JSON data) EXACTLY match the official PDF policy document. The PDF is the source of truth - the legal contract that the system must faithfully represent.

## CRITICAL RULES
1. Output ONLY valid JSON matching the schema below - no markdown, no explanations, no preamble
2. The PDF document is the SOURCE OF TRUTH (legal contract)
3. The JSON data is what the SYSTEM STORED and must be verified against the PDF
4. Any mismatch means the system has incorrect data that could lead to wrong premium calculations, incorrect payouts, or contract violations
5. Your response must start with { and end with }

---

## INPUT STRUCTURE

### System's Stored Data (JSON - What the system currently has)
%s

This JSON represents what the system extracted/stored from the PDF. Your task is to verify every field is correct.

### Field Severity Classification

**CRITICAL FIELDS** (if wrong in system = financial/legal risk):
- base_policy.fix_premium_amount (wrong amount = incorrect billing)
- base_policy.fix_payout_amount (wrong amount = incorrect claims)
- base_policy.coverage_duration_days (wrong duration = coverage gaps)
- base_policy.product_name (wrong product = misclassification)
- base_policy.crop_type (wrong crop = wrong risk assessment)
- base_policy.coverage_currency (wrong currency = calculation errors)
- triggers[*].conditions[*].threshold_value (wrong threshold = false triggers)
- triggers[*].conditions[*].threshold_operator (wrong operator = wrong trigger logic)

**IMPORTANT FIELDS** (if wrong = operational issues):
- base_policy.premium_base_rate (affects premium calculation)
- base_policy.payout_base_rate (affects payout calculation)
- base_policy.is_per_hectare (changes calculation method)
- base_policy.is_payout_per_hectare (changes payout calculation)
- base_policy.over_threshold_multiplier (affects payout scaling)
- base_policy.enrollment_start_day (wrong enrollment window)
- base_policy.enrollment_end_day (wrong enrollment window)
- base_policy.insurance_valid_from_day (wrong coverage start)
- base_policy.insurance_valid_to_day (wrong coverage end)
- base_policy.cancel_premium_rate (affects cancellation refunds)
- triggers[*].monitor_interval (wrong monitoring frequency)
- triggers[*].monitor_frequency_unit (wrong time unit)
- triggers[*].logical_operator (wrong condition combination logic)
- triggers[*].conditions[*].aggregation_function (wrong data aggregation)
- triggers[*].conditions[*].aggregation_window_days (wrong calculation window)
- triggers[*].conditions[*].validation_window_days (wrong validation period)
- triggers[*].conditions[*].consecutive_required (wrong trigger logic)

**METADATA FIELDS** (informational, not affecting calculations):
- base_policy.product_code
- base_policy.product_description
- base_policy.template_document_url
- base_policy.important_additional_information
- triggers[*].growth_stage
- triggers[*].blackout_periods

**OPTIONAL FIELDS** (may legitimately be null):
- base_policy.max_premium_payment_prolong
- base_policy.payout_cap
- base_policy.renewal_discount_rate
- base_policy.base_policy_invalid_date
- triggers[*].conditions[*].early_warning_threshold
- triggers[*].conditions[*].baseline_window_days
- triggers[*].conditions[*].baseline_function

---

## VALIDATION RULES

### Your Task for Each Field

1. **Locate in PDF**: Find where this field's value is stated in the PDF
2. **Extract PDF value**: Read the actual value from the PDF document
3. **Compare**: Check if system's JSON value matches PDF value
4. **Account for formatting**: Handle number formatting, percentages, currency symbols, language variations
5. **Flag discrepancies**: If system value ≠ PDF value, this is a mismatch

### Numeric Field Comparison

**Integer fields (int, int64):**
- Must match exactly
- PDF may format with separators: "500,000" or "500.000" or "500 000"
- System should store: 500000
- Strip all separators from PDF before comparing

**Float fields (float64):**
- Rates/multipliers: tolerance ±0.00001 (accounting for floating-point precision)
- PDF "5%" = system should store 0.05
- PDF "15.5%" = system should store 0.155
- PDF "0.05" or "5%" both mean system stores 0.05

**Currency amounts:**
- Must match exactly (integer values)
- PDF: "500,000 VND" = system: 500000
- PDF: "1,250,000 đ" = system: 1250000
- If PDF shows "500,000.00" and system has 500000 = PASS (decimal zeros ignored)

### String/Enum Field Comparison

System must store exact enum values. Map PDF text to correct enum:

**ThresholdOperator (system values):**
- System stores: "<" | PDF shows: "less than", "below", "under", "<", "dưới", "nhỏ hơn", "thấp hơn"
- System stores: ">" | PDF shows: "greater than", "above", "over", ">", "trên", "lớn hơn", "cao hơn"
- System stores: "<=" | PDF shows: "less than or equal", "at most", "<=", "≤", "nhỏ hơn hoặc bằng", "tối đa"
- System stores: ">=" | PDF shows: "greater than or equal", "at least", ">=", "≥", "lớn hơn hoặc bằng", "tối thiểu"
- System stores: "==" | PDF shows: "equal", "equals", "==", "=", "bằng"
- System stores: "!=" | PDF shows: "not equal", "!=", "≠", "khác"
- System stores: "change_gt" | PDF shows: "change greater than", "increase more than", "tăng trên", "thay đổi lớn hơn"
- System stores: "change_lt" | PDF shows: "change less than", "decrease more than", "giảm hơn", "thay đổi nhỏ hơn"

**LogicalOperator (system values):**
- System stores: "AND" | PDF shows: "and", "và", "&", "all of", "both", "tất cả", "đồng thời"
- System stores: "OR" | PDF shows: "or", "hoặc", "|", "any of", "either", "một trong", "bất kỳ"

**AggregationFunction (system values):**
- System stores: "sum" | PDF shows: "sum", "total", "tổng", "cộng dồn", "cumulative"
- System stores: "avg" | PDF shows: "average", "mean", "trung bình", "avg", "TB"
- System stores: "min" | PDF shows: "minimum", "min", "lowest", "tối thiểu", "thấp nhất"
- System stores: "max" | PDF shows: "maximum", "max", "highest", "tối đa", "cao nhất"
- System stores: "change" | PDF shows: "change", "difference", "thay đổi", "delta", "biến động"

**MonitorFrequency (system values):**
- System stores: "hour" | PDF shows: "hour", "hourly", "giờ", "per hour", "hàng giờ"
- System stores: "day" | PDF shows: "day", "daily", "ngày", "per day", "hàng ngày"
- System stores: "week" | PDF shows: "week", "weekly", "tuần", "per week", "hàng tuần"
- System stores: "month" | PDF shows: "month", "monthly", "tháng", "per month", "hàng tháng"
- System stores: "year" | PDF shows: "year", "yearly", "annual", "năm", "hàng năm"

**BasePolicyStatus (system values):**
- System stores: "draft" | PDF shows: "Draft", "Bản nháp", "Dự thảo"
- System stores: "active" | PDF shows: "Active", "Hoạt động", "Hiệu lực", "In Force"
- System stores: "archived" | PDF shows: "Archived", "Lưu trữ", "Hết hiệu lực"

**Boolean fields (system values):**
- System stores: true | PDF shows: "Yes", "True", "Có", "✓", "✔", "Enabled", "Được áp dụng"
- System stores: false | PDF shows: "No", "False", "Không", "✗", "✘", "Disabled", "Không áp dụng"

**Currency codes (system values):**
- System stores: "VND" | PDF shows: "VND", "đ", "₫", "Đồng", "Vietnamese Dong"
- System stores: "USD" | PDF shows: "USD", "$", "US Dollar", "Đô la Mỹ"

**Crop types:**
- System must match PDF exactly (case-insensitive)
- PDF: "Rice" or "Lúa" = system should store: "rice" or "Rice" (check case sensitivity)
- Common values: "rice", "corn", "coffee", "pepper", etc.

### Nested Structure Validation

**Arrays (triggers, conditions):**
- Count must match: If PDF describes 2 triggers, system must have 2 triggers
- Order matters: triggers[0] should match "Trigger 1" or "First condition" in PDF
- If PDF doesn't number triggers, match by semantic content

**JSON paths:**
- Use dot notation: base_policy.fix_premium_amount
- Use array indices: triggers[0].monitor_interval
- Nested arrays: triggers[0].conditions[1].threshold_value

### Date/Time Field Handling

**Day-of-year fields (enrollment_start_day, enrollment_end_day, etc.):**
- System stores: integer 1-365 (or 1-366 for leap years)
- PDF may show: "January 15", "15/01", "Ngày 15 tháng 1", "Day 15"
- Conversion: January 15 = day 15, December 31 = day 365
- If PDF shows month/day, convert to day-of-year

**Unix timestamp fields (base_policy_invalid_date, validation_timestamp):**
- System stores: integer (seconds since epoch)
- PDF may show: "2025-12-31", "31/12/2025", "December 31, 2025"
- Verify the date interpretation matches

### Special Calculation Fields

**is_per_hectare:**
- System stores: true → PDF must indicate "per hectare", "per ha", "/ha", "theo héc-ta", "mỗi héc-ta"
- System stores: false → PDF must indicate "per policy", "total", "fixed", "tổng", "cố định"

**consecutive_required:**
- System stores: true → PDF must indicate "consecutive", "liên tiếp", "continuous", "liên tục"
- System stores: false → PDF may indicate "cumulative", "total", "tổng cộng", "any occurrence"

**auto_renewal:**
- System stores: true → PDF must indicate "automatic renewal", "auto-renew", "tự động gia hạn"
- System stores: false → PDF must indicate "manual renewal", "không tự động", "needs renewal action"

### Null/Optional Field Handling

**If system has null:**
- PDF also has no value = PASS
- PDF has value = MISMATCH (system failed to extract)

**If system has value:**
- PDF has no value = MISMATCH (system added incorrect data)
- PDF has different value = MISMATCH (system extraction error)

---

## OUTPUT SCHEMA

Return a single valid JSON object with this exact structure:

{
  "id": "string (UUID v4: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx)",
  "base_policy_id": "string (copy exactly from input JSON base_policy.id)",
  "validation_timestamp": "integer (current Unix timestamp in seconds)",
  "validation_status": "string (MUST be: 'passed_ai' | 'failed' | 'warning')",
  "total_checks": "integer (total fields validated from system JSON)",
  "passed_checks": "integer (fields where system matches PDF)",
  "failed_checks": "integer (fields where system ≠ PDF)",
  "warning_count": "integer (ambiguous cases, optional fields)",
  
  "mismatches": {
    "<json_path>": {
      "json_value": "any (what system currently stores - INCORRECT)",
      "pdf_value": "any (what PDF actually says - CORRECT)",
      "severity": "string ('critical' | 'important' | 'metadata')",
      "field_type": "string ('int' | 'float64' | 'bool' | 'string' | 'enum')",
      "impact": "string (explain the consequence: 'Incorrect premium calculation' | 'Wrong payout trigger' | etc.)"
    }
  },
  
  "warnings": {
    "<json_path>": {
      "json_value": "any (system value)",
      "details": "string (reason: 'ambiguous_in_pdf' | 'multiple_possible_values' | 'formatting_unclear' | 'optional_field_absent')",
      "pdf_context": "string (relevant PDF text excerpt, max 200 chars)",
      "recommendation": "string (suggest manual review or clarification needed)"
    }
  },
  
  "recommendations": {
    "<category>": {
      "suggestion": "string (actionable fix for system data)",
      "affected_fields": ["array", "of", "json_paths"],
      "priority": "string ('high' | 'medium' | 'low')"
    }
  },
  
  "extracted_parameters": {
    "// Mirror of input JSON structure with values extracted from PDF"
    "// This shows what the system SHOULD have stored"
    "base_policy": {
      "fix_premium_amount": "value from PDF",
      "fix_payout_amount": "value from PDF"
    },
    "triggers": [
      {
        "monitor_interval": "value from PDF",
        "conditions": [
          {
            "threshold_value": "value from PDF"
          }
        ]
      }
    ]
  },
  
  "validated_by": "gemini-2.5-pro",
  "validation_notes": "string (summary: 'Validated X fields. Y match PDF, Z mismatches found. Critical issues: ...', max 500 chars)"
}

---

## VALIDATION STATUS LOGIC

Use this exact decision tree:

1. Count CRITICAL field mismatches:
   IF any CRITICAL field mismatch found:
     → validation_status = "failed"
     → System has incorrect data that will cause wrong calculations or contract violations
     → STOP

2. Count IMPORTANT field mismatches:
   IF more than 2 IMPORTANT field mismatches:
     → validation_status = "failed"
     → System has too many operational errors
     → STOP

3. Check for any mismatches or warnings:
   IF failed_checks > 0:
     → validation_status = "failed"
     → System data does not match PDF
   ELSE IF warning_count > 0:
     → validation_status = "warning"
     → System data might be incorrect, needs manual review
   ELSE:
     → validation_status = "passed_ai"
     → System data matches PDF document

---

## MISMATCH IMPACT DESCRIPTIONS

When you find a mismatch, include specific impact explanation:

**Premium/Payout amount mismatches:**
- Impact: "System will calculate incorrect premium/payout. PDF states [X], system has [Y], difference of [Z]."

**Threshold operator mismatches:**
- Impact: "Claim triggers will fire incorrectly. PDF requires [operator], system uses [operator], causing [false positives/negatives]."

**Duration/date mismatches:**
- Impact: "Coverage period incorrect. PDF specifies [X] days, system has [Y] days, creating [gap/overlap] of [Z] days."

**Rate/multiplier mismatches:**
- Impact: "Calculation formula incorrect. Using system rate of [X] instead of PDF rate [Y] will result in [higher/lower] premiums by [Z]%."

**Boolean logic mismatches:**
- Impact: "Calculation method wrong. PDF specifies [per hectare], system uses [fixed amount], causing incorrect scaling."

---

## EXTRACTION PRIORITY

When examining PDF, prioritize information in this order:

1. **Structured tables** (highest reliability)
2. **Numbered/bulleted lists**
3. **Section headers with values**
4. **Inline text within paragraphs**
5. **Footnotes or appendices** (lowest reliability)

If same field appears multiple times with different values in PDF:
- Use value from most authoritative section (usually main contract terms, not examples)
- Flag as WARNING noting the discrepancy
- Include both values in pdf_context

---

## EDGE CASES

**CASE 1: PDF value ambiguous or has multiple interpretations**
- Example: "Premium ranges from 500,000 to 1,000,000 VND" but system stores 500000
- Action: Add WARNING, note that system chose minimum value
- Recommendation: "Clarify if system should store minimum, maximum, or average"

**CASE 2: PDF uses formula/calculation, system stores result**
- Example: PDF says "Premium = Base Rate (5%) × Coverage Amount"
- System stores: premium_base_rate = 0.05
- Action: PASS if extracted rate matches, verify calculation logic is correct

**CASE 3: System array length ≠ PDF sections**
- Example: PDF describes 2 trigger conditions, system has 3 triggers
- Action: FAIL, add mismatch: "triggers array length mismatch"
- Impact: "System has extra/missing trigger conditions not in contract"

**CASE 4: Currency conversion or multiple currencies**
- PDF mentions both VND and USD
- System must store correct currency from the operative clause
- If PDF says "Premium: 500,000 VND (≈ $21 USD)" → system should store VND as coverage_currency

**CASE 5: Percentage formatting variations**
- PDF: "5%" or "0.05" or "5 percent"
- All should result in system storing: 0.05
- Check if system correctly normalized to decimal

**CASE 6: Date ambiguity**
- PDF: "Coverage starts on January 15"
- If no year specified, assume current/following year based on context
- Flag as WARNING if year ambiguous

**CASE 7: Missing critical fields in PDF**
- If CRITICAL field is in system but not found in PDF
- Action: FAIL with pdf_value = "NOT_FOUND"
- Impact: "System has value not authorized by contract document"

**CASE 8: PDF has value but system stores null**
- Example: PDF specifies product_code "AGR-001", system has null
- Action: Severity depends on field importance
- If IMPORTANT: FAIL
- If OPTIONAL/METADATA: WARNING

**CASE 9: Implicit vs explicit values**
- PDF: "Monitoring is performed daily" (implicit: monitor_frequency_unit = "day")
- System should store: "day"
- Extract implicit values and verify system captured correctly

**CASE 10: Vietnamese vs English terminology**
- PDF may use Vietnamese terms: "Lúa" (rice), "Héc-ta" (hectare)
- System may store English: "rice", or store Vietnamese
- Both are correct if semantically equivalent
- Check consistency: don't flag as mismatch if both valid

---

## QUALITY ASSURANCE CHECKLIST

Before outputting JSON, verify:

- [ ] validation_status is exactly one of: "passed_ai", "failed", "warning" (NOT "passed")
- [ ] total_checks = passed_checks + failed_checks + warning_count
- [ ] Every CRITICAL field from input JSON was checked
- [ ] Every mismatch has "impact" explanation
- [ ] extracted_parameters contains values FROM PDF (not from system JSON)
- [ ] All enum values in extracted_parameters match Go constants exactly
- [ ] UUID is valid v4 format
- [ ] validation_timestamp is current Unix epoch seconds (10 digits)
- [ ] No markdown formatting in output
- [ ] Response is valid JSON (starts with {, ends with })
- [ ] All json_path strings use correct notation
- [ ] If failed_checks > 0, mismatches object is not empty
- [ ] severity field uses only: "critical", "important", or "metadata"

---

## RESPONSE EXAMPLES

### Example 1: Critical Premium Amount Mismatch

**System JSON:** base_policy.fix_premium_amount = 500000
**PDF Content:** "Phí bảo hiểm cố định: 550,000 VND"

**Output excerpt:**
{
  "mismatches": {
    "base_policy.fix_premium_amount": {
      "json_value": 500000,
      "pdf_value": 550000,
      "severity": "critical",
      "field_type": "int",
      "impact": "System will undercharge premium by 50,000 VND per policy. PDF states 550,000 VND, system stores 500,000 VND."
    }
  },
  "failed_checks": 1,
  "validation_status": "failed",
  "recommendations": {
    "data_correction": {
      "suggestion": "Update base_policy.fix_premium_amount from 500000 to 550000 in database",
      "affected_fields": ["base_policy.fix_premium_amount"],
      "priority": "high"
    }
  }
}

### Example 2: Threshold Operator Wrong

**System JSON:** triggers[0].conditions[0].threshold_operator = ">"
**PDF Content:** "Claim is triggered when rainfall is below 100mm"

**Output excerpt:**
{
  "mismatches": {
    "triggers[0].conditions[0].threshold_operator": {
      "json_value": ">",
      "pdf_value": "<",
      "severity": "critical",
      "field_type": "enum",
      "impact": "Claim trigger logic is inverted. PDF specifies 'below' (<), system uses '>', causing claims to trigger at wrong conditions."
    }
  },
  "failed_checks": 1,
  "validation_status": "failed"
}

### Example 3: Percentage Correctly Stored

**System JSON:** base_policy.premium_base_rate = 0.05
**PDF Content:** "Tỷ lệ phí bảo hiểm cơ bản: 5%"

**Output excerpt:**
{
  "passed_checks": 1,
  "extracted_parameters": {
    "base_policy": {
      "premium_base_rate": 0.05
    }
  }
}
(Note: System correctly converted 5% to 0.05)

### Example 4: Array Length Mismatch

**System JSON:** triggers array has 2 elements
**PDF Content:** Describes 3 separate trigger conditions

**Output excerpt:**
{
  "mismatches": {
    "triggers": {
      "json_value": [2, "triggers"],
      "pdf_value": [3, "conditions described"],
      "severity": "critical",
      "field_type": "array",
      "impact": "System is missing one trigger condition from contract. PDF specifies 3 conditions, system only has 2."
    }
  },
  "failed_checks": 1,
  "validation_status": "failed",
  "recommendations": {
    "missing_trigger": {
      "suggestion": "Add third trigger condition as specified in PDF Section 4.3: 'Temperature exceeds 35°C for 5 consecutive days'",
      "affected_fields": ["triggers"],
      "priority": "high"
    }
  }
}

### Example 5: Optional Field Absent (Acceptable)

**System JSON:** base_policy.product_code = null
**PDF Content:** No product code mentioned

**Output excerpt:**
{
  "passed_checks": 1,
  "validation_notes": "Optional field product_code correctly null, not specified in PDF"
}

### Example 6: Boolean Mismatch

**System JSON:** base_policy.is_per_hectare = false
**PDF Content:** "Phí bảo hiểm được tính theo héc-ta" (Premium calculated per hectare)

**Output excerpt:**
{
  "mismatches": {
    "base_policy.is_per_hectare": {
      "json_value": false,
      "pdf_value": true,
      "severity": "critical",
      "field_type": "bool",
      "impact": "System will calculate premium as fixed amount instead of per-hectare, causing incorrect scaling for farm sizes."
    }
  },
  "failed_checks": 1,
  "validation_status": "failed"
}

---

## FINAL INSTRUCTION

Read the PDF document carefully. For each field in the system JSON, find its corresponding value in the PDF. Compare them. Report all mismatches with detailed impact analysis.

Your goal: Ensure the system operates exactly as the PDF contract states. Any deviation is a potential source of financial loss, legal liability, or customer disputes. Translate the response values into Vietnamese

BEGIN YOUR JSON OUTPUT NOW (start with opening brace):
`

// BuildRiskAnalysisPrompt constructs the comprehensive AI prompt for risk analysis
func BuildRiskAnalysisPrompt(
	farm models.Farm,
	farmPhotos []models.FarmPhoto, // Will include base64 image data
	farmPhotosData []string,
	monitoringData []models.FarmMonitoringData,
	trigger models.BasePolicyTrigger,
	conditions []models.BasePolicyTriggerCondition,
	dataSources map[string]models.DataSource, // keyed by data_source_id
	policy models.RegisteredPolicy,
) string {
	// Format farm photos with base64 data
	farmPhotosJSON := formatFarmPhotosWithImages(farmPhotos, farmPhotosData)

	// Format monitoring data grouped by parameter
	monitoringDataJSON := formatMonitoringDataGrouped(monitoringData)

	// Format trigger conditions with data source details
	conditionsJSON := formatConditionsWithDataSources(conditions, dataSources)

	// Format data sources list
	dataSourcesJSON := formatDataSources(dataSources)

	// Extract unique parameters being monitored
	parametersMonitored := extractUniqueParameters(monitoringData)

	// Current timestamp
	currentTimestamp := time.Now().Unix()

	prompt := fmt.Sprintf(`# Agricultural Insurance Risk Analysis Task - Multi-Parameter Analysis

## Context
You are analyzing a registered agricultural insurance policy for parametric crop insurance in Vietnam. Your role is to assess the risk level based on farm characteristics, historical monitoring data across MULTIPLE parameters (vegetation indices, weather data, derived metrics), and policy trigger configurations. This analysis supports insurance underwriting decisions.

## Task
Analyze the provided multi-parameter data and generate a comprehensive risk assessment that evaluates claim likelihood and fraud indicators.

---

## Input Data Structures

### 1. Farm Profile

**Farm Metadata:**
- Farm ID: %s
- Owner ID: %s
- Farm Name: %s
- Farm Code: %s
- Area (m²): %.2f
- Agro Polygon ID: %s

**Geographic Information:**
- Province: %s
- District: %s
- Commune: %s
- Full Address: %s
- Center Coordinates: %v
- Boundary Polygon: %v

**Crop Information:**
- Crop Type: %s
- Planting Date: %d (Unix timestamp)
- Expected Harvest Date: %d (Unix timestamp)
- Crop Type Verified: %t
- Crop Type Confidence: %.2f%%

**Land & Infrastructure:**
- Land Certificate Number: %s
- Land Ownership Verified: %t
- Land Ownership Verified At: %d
- Has Irrigation: %t
- Irrigation Type: %s
- Soil Type: %s
- Farm Status: %s

### 2. Latest Farm Photos (Including Base64 Image Data)

**Photo Collection:**
%s

**Instructions for Photo Analysis:**
- Each photo includes base64-encoded image data
- Photo types: crop, boundary, land_certificate, satellite, other
- Analyze visual indicators of farm condition, crop health, infrastructure quality
- Cross-reference photos with satellite data and reported farm characteristics
- Flag inconsistencies between photos and other data sources

### 3. Historical Monitoring Data (1 Year) - Multi-Parameter

**Parameters Being Monitored:** %s

**Time-Series Dataset (Grouped by Parameter):**
%s

**Data Structure Explanation:**
Each measurement contains:
- Parameter Name: The specific metric being measured
- Measured Value: The quantitative reading
- Unit: Measurement unit
- Measurement Timestamp: Unix timestamp of measurement
- Data Quality: "good", "acceptable", or "poor"
- Confidence Score: 0.0 to 1.0 reliability metric
- Component Data: Additional details specific to the parameter
- Cloud Cover Percentage: Relevant for satellite measurements
- Distance From Farm: Meters from measurement point (for weather stations)

**Analysis Instructions:**
For EACH parameter type, analyze:
- Statistical patterns (mean, median, std dev, min, max)
- Temporal trends (improving, stable, declining)
- Seasonal variations and cycles
- Anomalies and outliers
- Data completeness and quality distribution
- Cross-parameter correlations
- Alignment with crop growth stages

### 4. Data Source Metadata

**Data Sources Configuration:**
%s

**Key Information Per Source:**
- Data Source Type: weather/satellite/derived
- Parameter specifications and valid ranges
- Update frequency and spatial resolution
- Accuracy ratings and reliability
- Data provider information

### 5. Policy Trigger Configuration

**Base Trigger Settings:**
- Trigger ID: %s
- Logical Operator: %s (determines how multiple conditions combine)
- Growth Stage: %s
- Monitor Interval: %d
- Monitor Frequency Unit: %s
- Blackout Periods: %v

**Trigger Conditions (Total: %d):**
%s

**Condition Details Explanation:**
Each condition specifies:
- Which parameter is being monitored
- Threshold operator (<, >, <=, >=, ==, !=, change_gt, change_lt)
- Threshold value that triggers payout
- Aggregation function (sum/avg/min/max/change)
- Time window for aggregation
- Whether consecutive days are required
- Validation and baseline comparison settings

### 6. Registered Policy Details

**Policy Information:**
- Policy ID: %s
- Policy Number: %s
- Base Policy ID: %s
- Insurance Provider ID: %s
- Farmer ID: %s

**Coverage & Financial:**
- Coverage Amount: %.2f
- Coverage Start Date: %d (Unix timestamp)
- Coverage End Date: %d (Unix timestamp)
- Planting Date: %d (Unix timestamp)
- Area Multiplier: %.2f
- Total Farmer Premium: %.2f
- Premium Paid By Farmer: %t
- Premium Paid At: %d

**Data Complexity:**
- Data Complexity Score: %d
- Monthly Data Cost: %.2f
- Total Data Cost: %.2f

**Policy Status:**
- Status: %s
- Underwriting Status: %s

**Current Analysis Timestamp:** %d

---

## Risk Assessment Framework

### A. Farm Characteristic Analysis
Evaluate and score (0-100) the following factors:

**1. Geographic Risk (Weight: 20%%)**
- Historical climate patterns for Province: %s, District: %s
- Proximity to water sources (analyze from coordinates and irrigation data)
- Elevation and topography (derive from coordinates if possible)
- Known flood/drought zones for this region
- Distance from monitoring stations (check distance_from_farm_meters in monitoring data)

**2. Infrastructure Quality (Weight: 15%%)**
- Irrigation system: %t (Has Irrigation: %s)
- Land quality and soil type: %s
- Farm accessibility (analyze from location data)
- Infrastructure condition from photos (analyze photo data)
- Irrigation appropriateness for crop type: %s

**3. Crop Viability (Weight: 15%%)**
- Crop type: %s suitability for region: %s, %s
- Planting date: %d - assess appropriateness for season
- Crop type verification confidence: %.2f%% (verified: %t)
- Growth cycle alignment: planting %d to expected harvest %d vs coverage %d to %d
- Expected harvest date reasonableness

---

### B. Multi-Parameter Historical Performance Analysis (Weight: 30%%)

**Instructions:** Analyze ALL monitoring data grouped by parameter type. Calculate comprehensive statistics for each parameter and identify patterns, anomalies, and concerning trends.

**Key Analysis Tasks:**
1. **Vegetation Health Analysis** (if NDVI, NDMI, EVI data present)
   - Calculate mean, median, std dev across measurement period
   - Identify growth curve patterns and compare to normal crop phenology
   - Detect stress periods (NDVI < 0.4, NDMI < 0.2)
   - Analyze recent trends (last 30, 60, 90 days)
   - Flag declining trends before policy start date

2. **Weather Parameter Analysis** (analyze all weather parameters provided)
   - **Rainfall:** Total accumulation, drought periods (>14 days with <5mm), extreme events
   - **Temperature:** Mean, extremes, heat/cold stress days, Growing Degree Days
   - **Humidity:** Average levels, disease risk periods (>80%% sustained)
   - **Wind:** Average speeds, extreme events (>17 m/s)
   - **Other parameters:** Apply parameter-specific analysis

3. **Cross-Parameter Correlation Analysis**
   - Calculate correlations between vegetation indices and weather
   - Identify expected vs actual relationships
   - Flag contradictions (e.g., NDVI declining despite adequate rainfall)
   - Assess data consistency

4. **Data Quality Assessment**
   - Calculate completeness percentage per parameter
   - Quality distribution (good/acceptable/poor)
   - Identify data gaps, especially during critical growth stages
   - Average confidence scores
   - Cloud cover impact on satellite data

**Parameter-Specific Thresholds:**

*Vegetation Indices:*
- NDVI healthy range for %s: Refer to crop-specific benchmarks
- NDMI water stress threshold: < 0.2
- Stress detection: >30%% decline over 30 days

*Weather Parameters:*
- Rainfall for %s: Typical seasonal requirements
- Temperature stress: Heat >35°C, Cold <15°C for rice
- Drought definition: <75%% normal rainfall (moderate), <50%% (severe)

---

### C. Trigger Risk Analysis (Weight: 20%%)

**Critical Task:** Simulate ALL trigger conditions using the 1-year historical monitoring data.

**For EACH of the %d trigger conditions:**

1. **Extract trigger parameters:**
   - Parameter being monitored
   - Threshold operator and value
   - Aggregation function and window
   - Consecutive requirement

2. **Historical Simulation:**
   - Apply aggregation function over specified window
   - Scan through entire monitoring dataset
   - Count how many times threshold would have been breached
   - Record specific dates and values at breach
   - Calculate margin to threshold (average distance)

3. **Risk Assessment:**
   - **High Risk:** >4 historical triggers in past year
   - **Moderate Risk:** 2-4 historical triggers
   - **Low Risk:** 0-1 historical triggers
   - **Current Proximity:** How close are current conditions to trigger?

4. **Multi-Condition Logic Analysis:**
   - Logical operator is: %s
   - If AND: Calculate joint probability (all conditions met simultaneously)
   - If OR: Calculate union probability (any condition met)
   - Assess overall trigger activation risk

**Trigger Sensitivity Assessment:**
- Too Loose: Triggers frequently (>3 times/year) → Premium underpriced
- Well-Calibrated: Triggers rarely (0-1 times/year) → Appropriate protection
- Too Tight: Never triggers despite adverse conditions → Ineffective policy

---

### D. Fraud Risk Assessment (Weight: 10%%)

**Multi-Parameter Fraud Detection Framework:**

**1. Timing Fraud Indicators (CRITICAL):**
- Recent planting (<30 days before policy start %d): +20 points
- Policy start coincides with known adverse conditions: +15 points
- Application timing suspicious relative to seasonal patterns: +10 points

**2. Vegetation Health Fraud Indicators:**
Analyze NDVI/NDMI trends in 60 days before policy start (%d):
- NDVI declining >30%% in 60 days pre-policy: +25 points
- NDVI below crop-specific threshold pre-policy: +15 points
- Multiple vegetation indices showing stress: +10 points
- Sudden NDVI drop right before policy start: +20 points

**3. Weather-Based Fraud Indicators:**
Analyze weather data around policy start:
- Existing drought (>30 days no rain) at policy start: +20 points
- Trigger threshold close to current conditions (<10%% margin): +25 points
- Multiple weather parameters near trigger simultaneously: +15 points

**4. Documentation Fraud Indicators:**
- Unverified land certificate (%t): +15 points
- Unverified crop type (confidence %.2f%%): +20 if <50%%
- Missing or poor-quality farm photos: +10 points
- Boundary inconsistent with satellite imagery: +25 points

**5. Structural Fraud Indicators:**
- Coverage amount (%.2f) >> expected crop value: +20 if >2x
- Premium very low relative to trigger risk: +15 points
- All triggers set to activate easily: +25 points

**6. Cross-Parameter Inconsistencies:**
- Data contradictions between parameters: +15 points per contradiction
- Suspicious alignment (all parameters point to payout): +15 points

**Fraud Score Calculation:** Sum all applicable points
- >50: CRITICAL fraud risk
- 30-50: HIGH fraud risk
- 15-29: MODERATE fraud risk
- <15: LOW fraud risk

---

## Output Format Requirements

Generate a JSON response matching the RegisteredPolicyRiskAnalysis model structure:

{
  "analysis_status": "passed_ai" | "failed" | "warning",
  "analysis_type": "ai_model",
  "analysis_source": "Agricultural Risk AI Analyzer v2.0 - Multi-Parameter",
  "analysis_timestamp": <current_unix_timestamp>,
  "overall_risk_score": <float 0-100>,
  "overall_risk_level": "low" | "medium" | "high" | "critical",
  
  "identified_risks": {
    // Detailed risk breakdown with scores, levels, and specific factors with evidence
    // See full schema in documentation
  },
  
  "recommendations": {
    "underwriting_decision": {
      "recommendation": "approve" | "approve_with_conditions" | "reject" | "request_additional_info",
      "confidence": <0-100>,
      "reasoning": "Multi-paragraph explanation"
    },
    "suggested_actions": [...],
    "premium_adjustment": {...},
    "monitoring_recommendations": {...},
    "required_verifications": [...],
    "trigger_adjustments": {...}
  },
  
  "raw_output": {
    // Comprehensive statistical analysis and technical details
    // See full schema in documentation
  },
  
  "analysis_notes": "Executive summary (2-3 sentences)"
}

---

## Risk Scoring Guidelines

### Overall Risk Score Calculation (0-100)
Lower score = lower risk (better)
Higher score = higher risk (concerning)

**Weighted Components:**
1. Geographic Risk (20%%): 0-20 points
2. Infrastructure Quality (15%%): 0-15 points
3. Crop Viability (15%%): 0-15 points
4. Historical Performance (30%%): 0-30 points
5. Trigger Activation Risk (20%%): 0-20 points
6. Fraud Indicators (10%%): 0-10 points

**Risk Levels:**
- 0-25: LOW
- 26-50: MEDIUM
- 51-75: HIGH
- 76-100: CRITICAL

**Analysis Status Decision:**
- "passed_ai": Risk ≤50 AND fraud <40 AND no critical flags
- "warning": Risk 51-75 OR fraud 40-60
- "failed": Risk >75 OR fraud >60 OR critical indicators

---

## Vietnam-Specific Context

**Regional Climate:**
- Province: %s, District: %s, Commune: %s
- Assess regional vulnerability based on location

**Seasonal Context:**
- Planting Date: %d
- Coverage Period: %d to %d
- Determine dry/wet season alignment

**Crop-Specific Requirements for %s:**
- Water requirements
- Temperature tolerances
- Growth stage durations
- Typical NDVI/NDMI ranges

**Data Quality Expectations:**
- Satellite: 60-80%% cloud cover during monsoon
- Weather stations: Variable density
- Sentinel-2 NDVI: 5-day revisit

---

## Critical Analysis Constraints

1. **Evidence-Based:** Every claim must cite specific data values
2. **Conservative Bias:** When uncertain, assess higher risk
3. **Completeness Check:** Flag if ANY parameter has >20%% gaps
4. **Verification Triggers:**
   - Fraud score >20
   - Cross-parameter contradictions
   - Data quality <70%% for critical parameters
   - Historical trigger activations >2 times/year
5. **No Assumptions:** Only analyze provided data

---

## Processing Instructions

1. Load and validate all input data
2. Calculate statistical analysis for EACH parameter
3. Perform temporal analysis (trends, seasonality, anomalies)
4. Execute cross-correlation analysis between parameters
5. Simulate ALL %d trigger conditions historically
6. Apply fraud detection scoring framework
7. Aggregate weighted risk components
8. Generate specific, actionable recommendations
9. Compile complete JSON output
10. Write executive summary in analysis_notes

**Quality Checks:**
- All scores between 0-100
- All evidence cites specific data points
- Recommendations are actionable and specific
- JSON structure matches schema exactly
- No contradictions in findings

---

## Final Reminders

- This analysis affects farmers' livelihoods and insurance solvency
- Multi-parameter analysis provides robustness through cross-validation
- Be thorough, objective, and transparent about confidence levels
- Flag limitations and data gaps explicitly
- Prioritize fraud detection - it protects both farmers and insurers
- When in doubt, recommend manual verification
- Translate response value to Vietnamese

**Current Analysis Parameters:**
- Farm: %s (ID: %s)
- Policy: %s (ID: %s)
- Parameters Monitored: %s
- Trigger Conditions: %d
- Monitoring Data Points: %d
- Analysis Timestamp: %d`,

		// Farm Profile (1-20)
		farm.ID,                                       // 1
		farm.OwnerID,                                  // 2
		stringPtrOrEmpty(farm.FarmName),               // 3
		stringPtrOrEmpty(farm.FarmCode),               // 4
		farm.AreaSqm,                                  // 5
		farm.AgroPolygonID,                            // 6
		stringPtrOrEmpty(farm.Province),               // 7
		stringPtrOrEmpty(farm.District),               // 8
		stringPtrOrEmpty(farm.Commune),                // 9
		stringPtrOrEmpty(farm.Address),                // 10
		formatGeoJSONPoint(farm.CenterLocation),       // 11
		formatGeoJSONPolygon(farm.Boundary),           // 12
		farm.CropType,                                 // 13
		int64PtrOrZero(farm.PlantingDate),             // 14
		int64PtrOrZero(farm.ExpectedHarvestDate),      // 15
		farm.CropTypeVerified,                         // 16
		float64PtrOrZero(farm.CropTypeConfidence)*100, // 17
		stringPtrOrEmpty(farm.LandCertificateNumber),  // 18
		farm.LandOwnershipVerified,                    // 19
		int64PtrOrZero(farm.LandOwnershipVerifiedAt),  // 20
		farm.HasIrrigation,                            // 21
		stringPtrOrEmpty(farm.IrrigationType),         // 22
		stringPtrOrEmpty(farm.SoilType),               // 23
		farm.Status,                                   // 24

		// Farm Photos (25)
		farmPhotosJSON, // 25

		// Monitoring Data (26-27)
		strings.Join(parametersMonitored, ", "), // 26
		monitoringDataJSON,                      // 27

		// Data Sources (28)
		dataSourcesJSON, // 28

		// Trigger Configuration (29-35)
		trigger.ID,                             // 29
		trigger.LogicalOperator,                // 30
		stringPtrOrEmpty(trigger.GrowthStage),  // 31
		trigger.MonitorInterval,                // 32
		trigger.MonitorFrequencyUnit,           // 33
		formatJSONMap(trigger.BlackoutPeriods), // 34
		len(conditions),                        // 35

		// Conditions Details (36)
		conditionsJSON, // 36

		// Registered Policy (37-57)
		policy.ID,                            // 37
		policy.PolicyNumber,                  // 38
		policy.BasePolicyID,                  // 39
		policy.InsuranceProviderID,           // 40
		policy.FarmerID,                      // 41
		policy.CoverageAmount,                // 42
		policy.CoverageStartDate,             // 43
		policy.CoverageEndDate,               // 44
		policy.PlantingDate,                  // 45
		policy.AreaMultiplier,                // 46
		policy.TotalFarmerPremium,            // 47
		policy.PremiumPaidByFarmer,           // 48
		int64PtrOrZero(policy.PremiumPaidAt), // 49
		policy.DataComplexityScore,           // 50
		policy.MonthlyDataCost,               // 51
		policy.TotalDataCost,                 // 52
		policy.Status,                        // 53
		policy.UnderwritingStatus,            // 54
		currentTimestamp,                     // 55

		// Geographic context repeated for analysis sections (56-58)
		stringPtrOrEmpty(farm.Province),               // 56
		stringPtrOrEmpty(farm.District),               // 57
		farm.HasIrrigation,                            // 58
		stringPtrOrEmpty(farm.IrrigationType),         // 59
		stringPtrOrEmpty(farm.SoilType),               // 60
		farm.CropType,                                 // 61
		stringPtrOrEmpty(farm.Province),               // 62
		stringPtrOrEmpty(farm.District),               // 63
		int64PtrOrZero(farm.PlantingDate),             // 64
		int64PtrOrZero(farm.PlantingDate),             // 65
		int64PtrOrZero(farm.ExpectedHarvestDate),      // 66
		policy.CoverageStartDate,                      // 67
		policy.CoverageEndDate,                        // 68
		farm.CropType,                                 // 69
		float64PtrOrZero(farm.CropTypeConfidence)*100, // 70
		farm.CropTypeVerified,                         // 71

		// Trigger analysis context (72-73)
		len(conditions),         // 72
		trigger.LogicalOperator, // 73

		// Fraud detection context (74-79)
		policy.CoverageStartDate,                      // 74
		policy.CoverageStartDate,                      // 75
		farm.LandOwnershipVerified,                    // 76
		float64PtrOrZero(farm.CropTypeConfidence)*100, // 77
		policy.CoverageAmount,                         // 78

		// Vietnam context (79-84)
		stringPtrOrEmpty(farm.Province),   // 79
		stringPtrOrEmpty(farm.District),   // 80
		stringPtrOrEmpty(farm.Commune),    // 81
		int64PtrOrZero(farm.PlantingDate), // 82
		policy.CoverageStartDate,          // 83
		policy.CoverageEndDate,            // 84
		farm.CropType,                     // 85

		// Final context summary (86-91)
		stringPtrOrEmpty(farm.FarmName),         // 86
		farm.ID,                                 // 87
		policy.PolicyNumber,                     // 88
		policy.ID,                               // 89
		strings.Join(parametersMonitored, ", "), // 90
		len(conditions),                         // 91
		len(monitoringData),                     // 92
		currentTimestamp,                        // 93
	)

	return prompt
}

// Helper functions

func formatFarmPhotosWithImages(photos []models.FarmPhoto, imageData []string) string {
	if len(photos) == 0 {
		return "No farm photos available."
	}

	if len(photos) != len(imageData) {
		return fmt.Sprintf("Photo data mismatch: %d photos but %d image data entries.", len(photos), len(imageData))
	}

	var builder strings.Builder
	builder.WriteString("[\n")

	for i, photo := range photos {
		// Escape JSON strings properly
		escapedPhotoURL, _ := json.Marshal(photo.PhotoURL)
		escapedImageData, _ := json.Marshal(imageData[i])

		photoJSON := fmt.Sprintf(`  {
    "id": "%s",
    "farm_id": "%s",
    "photo_type": "%s",
    "taken_at": %d,
    "photo_url": %s,
    "image_base64": %s,
    "created_at": "%s"
  }`,
			photo.ID,
			photo.FarmID,
			photo.PhotoType,
			int64PtrOrZero(photo.TakenAt),
			string(escapedPhotoURL),
			string(escapedImageData),
			photo.CreatedAt.Format(time.RFC3339),
		)

		builder.WriteString(photoJSON)
		if i < len(photos)-1 {
			builder.WriteString(",\n")
		}
	}

	builder.WriteString("\n]")
	return builder.String()
}

func formatMonitoringDataGrouped(data []models.FarmMonitoringData) string {
	if len(data) == 0 {
		return "No monitoring data available."
	}

	// Group by parameter name
	grouped := make(map[models.DataSourceParameterName][]models.FarmMonitoringData)
	for _, d := range data {
		grouped[d.ParameterName] = append(grouped[d.ParameterName], d)
	}

	var builder strings.Builder
	builder.WriteString("{\n")

	paramCount := 0
	totalParams := len(grouped)

	for paramName, measurements := range grouped {
		builder.WriteString(fmt.Sprintf(`  "%s": {
    "total_measurements": %d,
    "unit": "%s",
    "measurements": [
`,
			paramName,
			len(measurements),
			stringPtrOrEmpty(measurements[0].Unit),
		))

		for i, m := range measurements {
			measurementJSON := fmt.Sprintf(`      {
        "measured_value": %.4f,
        "measurement_timestamp": %d,
        "data_quality": "%s",
        "confidence_score": %.4f,
        "component_data": %s,
        "cloud_cover_percentage": %.2f,
        "distance_from_farm_meters": %.2f
      }`,
				m.MeasuredValue,
				m.MeasurementTimestamp,
				m.DataQuality,
				float64PtrOrZero(m.ConfidenceScore),
				formatJSONMap(m.ComponentData),
				float64PtrOrZero(m.CloudCoverPercentage),
				float64PtrOrZero(m.DistanceFromFarmMeters),
			)

			builder.WriteString(measurementJSON)
			if i < len(measurements)-1 {
				builder.WriteString(",\n")
			}
		}

		builder.WriteString("\n    ]\n  }")
		paramCount++
		if paramCount < totalParams {
			builder.WriteString(",\n")
		}
	}

	builder.WriteString("\n}")
	return builder.String()
}

func formatConditionsWithDataSources(
	conditions []models.BasePolicyTriggerCondition,
	dataSources map[string]models.DataSource,
) string {
	if len(conditions) == 0 {
		return "No trigger conditions configured."
	}

	var builder strings.Builder
	builder.WriteString("[\n")

	for i, cond := range conditions {
		ds, exists := dataSources[cond.DataSourceID.String()]
		paramName := "unknown"
		unit := "unknown"
		dataSourceWarning := ""
		if exists {
			paramName = string(ds.ParameterName)
			if ds.Unit != nil {
				unit = *ds.Unit
			}
		} else {
			dataSourceWarning = fmt.Sprintf("WARNING: Data source %s not found in provided map", cond.DataSourceID.String())
		}

		condJSON := fmt.Sprintf(`  {
    "condition_id": "%s",
    "condition_order": %d,
    "parameter_name": "%s",
    "parameter_unit": "%s",
    "threshold_operator": "%s",
    "threshold_value": %.4f,
    "early_warning_threshold": %.4f,
    "aggregation_function": "%s",
    "aggregation_window_days": %d,
    "consecutive_required": %t,
    "validation_window_days": %d,
    "baseline_window_days": %d,
    "baseline_function": "%s",
    "include_component": %t,
    "data_cost": %.2f,
    "data_source_warning": "%s"
  }`,
			cond.ID,
			cond.ConditionOrder,
			paramName,
			unit,
			cond.ThresholdOperator,
			cond.ThresholdValue,
			float64PtrOrZero(cond.EarlyWarningThreshold),
			cond.AggregationFunction,
			cond.AggregationWindowDays,
			cond.ConsecutiveRequired,
			cond.ValidationWindowDays,
			intPtrOrZero(cond.BaselineWindowDays),
			aggregationFunctionPtrOrEmpty(cond.BaselineFunction),
			cond.IncludeComponent,
			cond.CalculatedCost,
			dataSourceWarning,
		)

		builder.WriteString(condJSON)
		if i < len(conditions)-1 {
			builder.WriteString(",\n")
		}
	}

	builder.WriteString("\n]")
	return builder.String()
}

func formatDataSources(dataSources map[string]models.DataSource) string {
	if len(dataSources) == 0 {
		return "No data sources configured."
	}

	var builder strings.Builder
	builder.WriteString("[\n")

	count := 0
	total := len(dataSources)

	for _, ds := range dataSources {
		dsJSON := fmt.Sprintf(`  {
    "id": "%s",
    "data_source_type": "%s",
    "parameter_name": "%s",
    "parameter_type": "%s",
    "unit": "%s",
    "min_value": %.4f,
    "max_value": %.4f,
    "update_frequency": "%s",
    "spatial_resolution": "%s",
    "accuracy_rating": %.2f,
    "data_provider": "%s",
    "api_endpoint": "%s"
  }`,
			ds.ID,
			ds.DataSource,
			ds.ParameterName,
			ds.ParameterType,
			stringPtrOrEmpty(ds.Unit),
			float64PtrOrZero(ds.MinValue),
			float64PtrOrZero(ds.MaxValue),
			stringPtrOrEmpty(ds.UpdateFrequency),
			stringPtrOrEmpty(ds.SpatialResolution),
			float64PtrOrZero(ds.AccuracyRating),
			stringPtrOrEmpty(ds.DataProvider),
			stringPtrOrEmpty(ds.APIEndpoint),
		)

		builder.WriteString(dsJSON)
		count++
		if count < total {
			builder.WriteString(",\n")
		}
	}

	builder.WriteString("\n]")
	return builder.String()
}

func extractUniqueParameters(data []models.FarmMonitoringData) []string {
	paramMap := make(map[string]bool)
	for _, d := range data {
		paramMap[string(d.ParameterName)] = true
	}

	params := make([]string, 0, len(paramMap))
	for param := range paramMap {
		params = append(params, param)
	}
	return params
}

// Utility helper functions

func stringPtrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func int64PtrOrZero(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func intPtrOrZero(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func float64PtrOrZero(f *float64) float64 {
	if f == nil {
		return 0.0
	}
	return *f
}

func aggregationFunctionPtrOrEmpty(af *models.AggregationFunction) string {
	if af == nil {
		return ""
	}
	return string(*af)
}

func formatGeoJSONPoint(point *models.GeoJSONPoint) string {
	if point == nil {
		return "null"
	}
	b, _ := json.Marshal(point)
	return string(b)
}

func formatGeoJSONPolygon(polygon *models.GeoJSONPolygon) string {
	if polygon == nil {
		return "null"
	}
	b, _ := json.Marshal(polygon)
	return string(b)
}

func formatJSONMap(m map[string]interface{}) string {
	if m == nil || len(m) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(m)
	return string(b)
}
