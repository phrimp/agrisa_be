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
