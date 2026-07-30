package sanitizer

import (
	"reflect"
)

// SanitizeFields 遍历结构体字段，执行以下清洗操作：
// 1. 将所有值为 "" 的 *string 字段置为 nil。
// 2. 将所有值为 0 的 *int, *int64 等整型指针字段置为 nil。
// entity 必须是一个指向结构体的指针。
func SanitizeFields(entity interface{}) {
	val := reflect.ValueOf(entity)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}

		// 处理指针类型
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				continue
			}
			elem := field.Elem()

			// 1. 处理 *string
			if elem.Kind() == reflect.String {
				if elem.String() == "" {
					field.Set(reflect.Zero(field.Type()))
				}
			}

			// 2. 处理 *int, *int64 等整型
			if isIntKind(elem.Kind()) {
				if elem.Int() == 0 {
					field.Set(reflect.Zero(field.Type()))
				}
			}
		}
	}
}

func isIntKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}
