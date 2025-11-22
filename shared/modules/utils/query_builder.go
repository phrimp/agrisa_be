package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

type QueryBuildResult struct {
	Query string
	Args  []interface{}
}

// FieldTransformer định nghĩa cách transform một field đặc biệt
type FieldTransformer struct {
	// SQLFunc là hàm SQL cần wrap giá trị, ví dụ: "ST_GeomFromText"
	SQLFunc string
	// ConvertValue là hàm để convert giá trị trước khi truyền vào SQL
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

// BuildDynamicUpdateQuery xây dựng câu query UPDATE động
// Parameters:
//   - tableName: tên bảng cần update
//   - updateData: map chứa các field và giá trị cần update
//   - allowedFields: map các field được phép update
//   - arrayFields: map các field có kiểu array
//   - specialFields: map các field cần xử lý đặc biệt (PostGIS, JSON functions, etc.)
//   - whereField: tên field dùng trong WHERE clause
//   - whereValue: giá trị cho WHERE clause
//   - autoAddUpdatedAt: tự động thêm updated_at nếu true
func BuildDynamicUpdateQuery(
	tableName string,
	updateData map[string]interface{},
	allowedFields map[string]bool,
	arrayFields map[string]bool,
	specialFields map[string]*FieldTransformer,
	whereField string,
	whereValue interface{},
	autoAddUpdatedAt bool,
) (*QueryBuildResult, error) {
	setClauses := []string{}
	args := []interface{}{}
	argPosition := 1

	// Duyệt qua các field cần update
	for field, value := range updateData {
		// Kiểm tra field có được phép update không
		if !allowedFields[field] {
			return nil, fmt.Errorf("field %s is not allowed to be updated", field)
		}

		// Xử lý các field đặc biệt (PostGIS, JSON functions, etc.)
		if transformer, isSpecial := specialFields[field]; isSpecial {
			var processedValue interface{}
			var err error

			// Nếu có hàm convert, dùng hàm đó
			if transformer.ConvertValue != nil {
				processedValue, err = transformer.ConvertValue(value)
				if err != nil {
					return nil, fmt.Errorf("error converting field %s: %v", field, err)
				}
			} else {
				processedValue = value
			}

			// Tạo SET clause với SQL function
			if transformer.SQLFunc != "" {
				setClauses = append(setClauses, fmt.Sprintf("%s = %s($%d)", field, transformer.SQLFunc, argPosition))
			} else {
				setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argPosition))
			}
			args = append(args, processedValue)
			argPosition++
			continue
		}

		// Xử lý các field có kiểu array
		if arrayFields[field] {
			// Chuyển đổi slice interface{} thành []string
			if arr, ok := value.([]interface{}); ok {
				strArr := make([]string, len(arr))
				for i, v := range arr {
					strArr[i] = fmt.Sprintf("%v", v)
				}
				setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argPosition))
				args = append(args, pq.Array(strArr))
				argPosition++
			} else {
				return nil, fmt.Errorf("field %s should be an array", field)
			}
		} else {
			// Xử lý các field thông thường
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argPosition))
			args = append(args, value)
			argPosition++
		}
	}

	// Kiểm tra có field nào để update không
	if len(setClauses) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Tự động thêm updated_at nếu cần
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

	// Thêm giá trị cho WHERE clause
	args = append(args, whereValue)

	// Xây dựng query cuối cùng
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
