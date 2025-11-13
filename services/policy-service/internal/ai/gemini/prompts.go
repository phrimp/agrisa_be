package gemini

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
