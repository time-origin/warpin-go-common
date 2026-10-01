package database

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// CheckUniqueConstraints 检查实体中带有 uniqueIndex 标签的字段是否在数据库中已存在。
// 如果存在，返回格式为 "comment+已存在" 的错误。
// excludeID 用于更新操作时排除自身 ID。
func CheckUniqueConstraints(ctx context.Context, db *gorm.DB, entity interface{}, excludeID int64) error {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 获取 gorm 标签
		gormTag := fieldType.Tag.Get("gorm")
		if gormTag == "" {
			continue
		}

		// 检查是否包含 uniqueIndex
		if strings.Contains(gormTag, "uniqueIndex") {
			// 获取字段值
			var value interface{}
			if field.Kind() == reflect.Ptr {
				if field.IsNil() {
					continue // 如果指针为空，跳过检查
				}
				value = field.Elem().Interface()
			} else {
				value = field.Interface()
			}

			// 如果值为空字符串或零值，通常不需要检查唯一性（取决于业务，这里假设空值不参与唯一性检查）
			if isZero(value) {
				continue
			}

			// 获取数据库列名
			columnName := getColumnName(fieldType)
			if columnName == "" {
				continue
			}

			// 获取 comment
			comment := getComment(gormTag)
			if comment == "" {
				comment = fieldType.Name // 如果没有 comment，使用字段名兜底
			}

			// 构建查询
			// 注意：这里需要创建一个新的实例来作为 Model，避免污染传入的 entity
			// 或者直接使用 entity 的类型
			modelType := reflect.New(typ).Interface()
			query := db.WithContext(ctx).Model(modelType).Where(fmt.Sprintf("%s = ?", columnName), value)
			if excludeID > 0 {
				query = query.Where("id != ?", excludeID)
			}

			var count int64
			if err := query.Count(&count).Error; err != nil {
				return err
			}

			if count > 0 {
				return fmt.Errorf("%s已存在", comment)
			}
		}
	}

	return nil
}

// isZero 判断值是否为零值
func isZero(v interface{}) bool {
	return reflect.DeepEqual(v, reflect.Zero(reflect.TypeOf(v)).Interface())
}

// getColumnName 尝试从 gorm 标签或 json 标签获取列名，默认转蛇形
func getColumnName(field reflect.StructField) string {
	gormTag := field.Tag.Get("gorm")
	// 尝试从 gorm:column:xxx 获取
	parts := strings.Split(gormTag, ";")
	for _, part := range parts {
		if strings.HasPrefix(part, "column:") {
			return strings.TrimPrefix(part, "column:")
		}
	}

	// 尝试从 json 标签获取
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		return strings.Split(jsonTag, ",")[0]
	}

	// 默认使用字段名转蛇形 (这里简单处理，假设 GORM 默认策略)
	return toSnakeCase(field.Name)
}

// getComment 从 gorm 标签中提取 comment
func getComment(tag string) string {
	parts := strings.Split(tag, ";")
	for _, part := range parts {
		if strings.HasPrefix(part, "comment:") {
			return strings.TrimPrefix(part, "comment:")
		}
	}
	return ""
}

// toSnakeCase 简单的驼峰转蛇形
func toSnakeCase(str string) string {
	var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
