package utils

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lib/pq"
)

type QueryBuildResult struct {
	Query string
	Args  []interface{}
}

// FieldTransformer defines how to transform a special field
type FieldTransformer struct {
	// SQLFunc is the SQL function to wrap the value, e.g., "ST_GeomFromText"
	SQLFunc string
	// ConvertValue is a function to convert the value before passing to SQL
	ConvertValue func(value interface{}) (interface{}, error)
}

// GeoJSONToWKT converts GeoJSON object to WKT (Well-Known Text) string
func GeoJSONToWKT(geoJSON interface{}) (string, error) {
	geoMap, ok := geoJSON.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid GeoJSON format")
	}

	geoType, ok := geoMap["type"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'type' in GeoJSON")
	}

	coordinates := geoMap["coordinates"]

	switch geoType {
	case "Point":
		coords, ok := coordinates.([]interface{})
		if !ok || len(coords) != 2 {
			return "", fmt.Errorf("invalid Point coordinates")
		}
		return fmt.Sprintf("POINT(%v %v)", coords[0], coords[1]), nil

	case "Polygon":
		rings, ok := coordinates.([]interface{})
		if !ok || len(rings) == 0 {
			return "", fmt.Errorf("invalid Polygon coordinates")
		}

		var ringStrings []string
		for _, ring := range rings {
			points, ok := ring.([]interface{})
			if !ok {
				return "", fmt.Errorf("invalid ring in Polygon")
			}

			var pointStrings []string
			for _, point := range points {
				coords, ok := point.([]interface{})
				if !ok || len(coords) != 2 {
					return "", fmt.Errorf("invalid point coordinates in Polygon")
				}
				pointStrings = append(pointStrings, fmt.Sprintf("%v %v", coords[0], coords[1]))
			}
			ringStrings = append(ringStrings, "("+strings.Join(pointStrings, ", ")+")")
		}
		return fmt.Sprintf("POLYGON(%s)", strings.Join(ringStrings, ", ")), nil

	case "LineString":
		points, ok := coordinates.([]interface{})
		if !ok {
			return "", fmt.Errorf("invalid LineString coordinates")
		}

		var pointStrings []string
		for _, point := range points {
			coords, ok := point.([]interface{})
			if !ok || len(coords) != 2 {
				return "", fmt.Errorf("invalid point coordinates in LineString")
			}
			pointStrings = append(pointStrings, fmt.Sprintf("%v %v", coords[0], coords[1]))
		}
		return fmt.Sprintf("LINESTRING(%s)", strings.Join(pointStrings, ", ")), nil

	default:
		return "", fmt.Errorf("unsupported GeoJSON type: %s", geoType)
	}
}

// BuildDynamicUpdateQuery builds a dynamic UPDATE query
// Parameters:
//   - tableName: the name of the table to update
//   - updateData: a map containing the fields and values to update
//   - allowedFields: a map of fields that are allowed to be updated
//   - arrayFields: a map of fields that have array types
//   - specialFields: a map of fields that require special handling (PostGIS, JSON functions, etc.)
//   - whereField: the field name used in the WHERE clause
//   - whereValue: the value for the WHERE clause
//   - autoAddUpdatedAt: automatically adds updated_at if true
func BuildDynamicUpdateQuery(
	tableName string,
	updateData map[string]interface{},
	allowedFields map[string]bool,
	arrayFields map[string]bool,
	specialFields map[string]*FieldTransformer,
	whereField string,
	whereValue interface{},
	autoAddUpdatedAt bool,
	updatedBy string,
	updatedByFieldName string,
) (*QueryBuildResult, error) {
	setClauses := []string{}
	args := []interface{}{}
	argPosition := 1

	// Iterate through fields to update
	for field, value := range updateData {
		// Check if the field is allowed to be updated
		if !allowedFields[field] {
			slog.Error("Field not allowed to be updated", "field", field)
			return nil, fmt.Errorf("field %s is not allowed to be updated", field)
		}

		// Handle special fields (PostGIS, JSON functions, etc.)
		if transformer, isSpecial := specialFields[field]; isSpecial {
			var processedValue interface{}
			var err error

			// If there is a convert function, use it
			if transformer.ConvertValue != nil {
				processedValue, err = transformer.ConvertValue(value)
				if err != nil {
					return nil, fmt.Errorf("error converting field %s: %v", field, err)
				}
			} else {
				processedValue = value
			}

			// Create SET clause with SQL function
			if transformer.SQLFunc != "" {
				setClauses = append(setClauses, fmt.Sprintf("%s = %s($%d)", field, transformer.SQLFunc, argPosition))
			} else {
				setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argPosition))
			}
			args = append(args, processedValue)
			argPosition++
			continue
		}

		// handle array fields
		if arrayFields[field] {
			// Convert slice of interface{} to []string
			if arr, ok := value.([]interface{}); ok {
				strArr := make([]string, len(arr))
				for i, v := range arr {
					strArr[i] = fmt.Sprintf("%v", v)
				}
				setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argPosition))
				args = append(args, pq.Array(strArr))
				argPosition++
			} else {
				slog.Error("Field should be an array", "field", field)
				return nil, fmt.Errorf("field %s should be an array", field)
			}
		} else {
			// handle regular fields
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argPosition))
			args = append(args, value)
			argPosition++
		}
	}

	// check if there are fields to update
	if len(setClauses) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// auto add updated_at if enabled
	if autoAddUpdatedAt {
		hasUpdatedAt := false
		for field := range updateData {
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
	}

	// add updated_by if provided
	if updatedBy != "" {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", updatedByFieldName, argPosition))
		args = append(args, updatedBy)
		argPosition++
	}

	// add where value
	args = append(args, whereValue)

	// build final query
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d",
		tableName,
		strings.Join(setClauses, ", "),
		whereField,
		argPosition,
	)

	return &QueryBuildResult{
		Query: query,
		Args:  args,
	}, nil
}

// BuildDinamicFilter
type Condition struct {
	Field    string      // Column name
	Operator string      // Operator: =, !=, >, <, >=, <=, LIKE, IN, BETWEEN, is_null, is_not_null
	Value    interface{} // Value (can be single value, slice for IN, or []interface{}{min, max} for BETWEEN)
	Logic    string      // AND or OR (used to connect to the next condition)
}

// OrderBy represents sorting
type OrderBy struct {
	Column    string // Column name
	Direction string // ASC or DESC
}

// QueryBuilder contains information to build a query
type QueryBuilder struct {
	TemplateQuery string
	Conditions    []Condition
	OrderBy       []string // Example: []string{"name ASC", "age DESC"}
}

// BuildQueryDynamicFilter creates a dynamic query with parameterized placeholders
func (qb *QueryBuilder) BuildQueryDynamicFilter() (string, []interface{}, error) {
	if qb.TemplateQuery == "" {
		return "", nil, fmt.Errorf("template query is required")
	}

	query := qb.TemplateQuery
	args := []interface{}{}
	paramIndex := 1

	// Build WHERE clause
	if len(qb.Conditions) > 0 {
		whereClause := " WHERE "
		whereParts := []string{}

		for i, cond := range qb.Conditions {
			var condStr string

			switch strings.ToUpper(cond.Operator) {
			case "=", "!=", ">", "<", ">=", "<=":
				condStr = fmt.Sprintf("%s %s $%d", cond.Field, cond.Operator, paramIndex)
				args = append(args, cond.Value)
				paramIndex++

			case "LIKE":
				condStr = fmt.Sprintf("%s LIKE $%d", cond.Field, paramIndex)
				args = append(args, cond.Value)
				paramIndex++

			case "IN":
				values, ok := cond.Value.([]interface{})
				if !ok {
					return "", nil, fmt.Errorf("IN operator requires []interface{} value for field %s", cond.Field)
				}
				if len(values) == 0 {
					return "", nil, fmt.Errorf("IN operator requires at least one value for field %s", cond.Field)
				}

				placeholders := []string{}
				for _, val := range values {
					placeholders = append(placeholders, fmt.Sprintf("$%d", paramIndex))
					args = append(args, val)
					paramIndex++
				}
				condStr = fmt.Sprintf("%s IN (%s)", cond.Field, strings.Join(placeholders, ", "))

			case "BETWEEN":
				values, ok := cond.Value.([]interface{})
				if !ok || len(values) != 2 {
					return "", nil, fmt.Errorf("BETWEEN operator requires []interface{}{min, max} for field %s", cond.Field)
				}
				condStr = fmt.Sprintf("%s BETWEEN $%d AND $%d", cond.Field, paramIndex, paramIndex+1)
				args = append(args, values[0], values[1])
				paramIndex += 2

			case "IS_NULL":
				condStr = fmt.Sprintf("%s IS NULL", cond.Field)

			case "IS_NOT_NULL":
				condStr = fmt.Sprintf("%s IS NOT NULL", cond.Field)

			default:
				return "", nil, fmt.Errorf("unsupported operator: %s", cond.Operator)
			}

			whereParts = append(whereParts, condStr)

			// Add logic operator (AND/OR) if not the last condition
			if i < len(qb.Conditions)-1 {
				logic := strings.ToUpper(strings.TrimSpace(cond.Logic))
				if logic == "" {
					logic = "AND" // Default is AND
				}
				if logic != "AND" && logic != "OR" {
					return "", nil, fmt.Errorf("invalid logic operator: %s (must be AND or OR)", cond.Logic)
				}
				whereParts = append(whereParts, logic)
			}
		}

		whereClause += strings.Join(whereParts, " ")
		query += whereClause
	}

	// Build ORDER BY clause
	if len(qb.OrderBy) > 0 {
		validOrders := []string{}
		for _, order := range qb.OrderBy {
			order = strings.TrimSpace(order)
			if order != "" {
				// Validate format: "column ASC" or "column DESC"
				parts := strings.Fields(order)
				if len(parts) == 2 {
					dir := strings.ToUpper(parts[1])
					if dir == "ASC" || dir == "DESC" {
						validOrders = append(validOrders, fmt.Sprintf("%s %s", parts[0], dir))
					} else {
						return "", nil, fmt.Errorf("invalid order direction: %s (must be ASC or DESC)", parts[1])
					}
				} else if len(parts) == 1 {
					// If no direction is provided, default to ASC
					validOrders = append(validOrders, fmt.Sprintf("%s ASC", parts[0]))
				} else {
					return "", nil, fmt.Errorf("invalid order format: %s", order)
				}
			}
		}

		if len(validOrders) > 0 {
			query += " ORDER BY " + strings.Join(validOrders, ", ")
		}
	}

	return query, args, nil
}

// func BuildUpdateQuery(
// 	updateProfileRequestBody map[string]interface{},
// 	allowedUpdateInsuranceProfileFields map[string]bool,
// 	arrayInsuranceProfileFields map[string]bool,
// 	criteriaField string,
// 	table string,
// ) (string, []interface{}, error) {
// 	setClauses := []string{}
// 	args := []interface{}{}
// 	argPosition := 1

// 	for field, value := range updateProfileRequestBody {

// 		if !allowedUpdateInsuranceProfileFields[field] {
// 			slog.Error("Field not allowed to be updated", "field", field)
// 			return "", nil, fmt.Errorf("bad request: field %s is not allowed to be updated", field)
// 		}

// 		if arrayInsuranceProfileFields[field] {

// 			if arr, ok := value.([]interface{}); ok {
// 				strArr := make([]string, len(arr))
// 				for i, v := range arr {
// 					strArr[i] = fmt.Sprintf("%v", v)
// 				}
// 				setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argPosition))
// 				args = append(args, pq.Array(strArr))
// 				argPosition++
// 			}
// 		} else {

// 			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argPosition))
// 			args = append(args, value)
// 			argPosition++
// 		}
// 	}

// 	if len(setClauses) == 0 {
// 		slog.Error("No fields to update for insurance partner ID", "partner_id", criteriaField)
// 		return "", nil, fmt.Errorf("bad request: no fields to update for insurance partner ID %s", criteriaField)
// 	}

// 	hasUpdatedAt := false
// 	for field := range updateProfileRequestBody {
// 		if field == "updated_at" {
// 			hasUpdatedAt = true
// 			break
// 		}
// 	}
// 	if !hasUpdatedAt {
// 		setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argPosition))
// 		args = append(args, time.Now())
// 		argPosition++
// 	}

// 	args = append(args, criteriaField)

// 	query := fmt.Sprintf(
// 		"UPDATE %s SET %s WHERE %s = $%d",
// 		table,
// 		strings.Join(setClauses, ", "),
// 		criteriaField,
// 		argPosition,
// 	)

// 	return query, args, nil
// }
