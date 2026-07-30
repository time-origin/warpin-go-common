package converter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/sqlc-dev/pqtype"
)

// --- Existing functions (for pointer inputs) ---

// NullStringFromPtr 将一个 string 指针转换为 sql.NullString。
func NullStringFromPtr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// NullInt64FromPtr 将一个 int64 指针转换为 sql.NullInt64。
func NullInt64FromPtr(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

// NullTimeFromPtr 将一个 time.Time 指针转换为 sql.NullTime。
func NullTimeFromPtr(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// NullRawMessageFromPtr 将一个 json.RawMessage 指针转换为 pqtype.NullRawMessage。
func NullRawMessageFromPtr(j *json.RawMessage) pqtype.NullRawMessage {
	if j == nil {
		return pqtype.NullRawMessage{}
	}
	return pqtype.NullRawMessage{RawMessage: *j, Valid: true}
}

// --- New functions (for non-pointer inputs to sql.Null*) ---

// ToNullString converts a string to sql.NullString. If the string is empty, Valid is false.
func ToNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// ToNullInt64 converts an int64 to sql.NullInt64. If the int64 is zero, Valid is false.
func ToNullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{Int64: i, Valid: i != 0}
}

// ToNullTime converts a time.Time to sql.NullTime. If the time.Time is its zero value, Valid is false.
func ToNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: !t.IsZero()}
}

// ToNullJSONB converts a map[string]interface{} to pqtype.NullRawMessage.
// If the map is nil or empty, Valid is false.
func ToNullJSONB(m map[string]interface{}) pqtype.NullRawMessage {
	if m == nil || len(m) == 0 {
		return pqtype.NullRawMessage{}
	}
	data, err := json.Marshal(m)
	if err != nil {
		// In a real application, you might want to log this error or return an error.
		// For this context, returning an invalid NullRawMessage is a reasonable default.
		return pqtype.NullRawMessage{}
	}
	return pqtype.NullRawMessage{RawMessage: json.RawMessage(data), Valid: true}
}

// --- New functions (for sql.Null* to plain types) ---

// FromNullString converts sql.NullString to string. Returns empty string if not valid.
func FromNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// FromNullInt64 converts sql.NullInt64 to int64. Returns 0 if not valid.
func FromNullInt64(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}

// FromNullInt64Ptr converts sql.NullInt64 to *int64. Returns nil if not valid.
func FromNullInt64Ptr(ni sql.NullInt64) *int64 {
	if ni.Valid {
		return &ni.Int64
	}
	return nil
}

// FromNullTime converts sql.NullTime to *time.Time. Returns nil if not valid.
func FromNullTime(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

// FromNullBool converts sql.NullBool to *bool. Returns nil if not valid.
func FromNullBool(nb sql.NullBool) *bool {
	if nb.Valid {
		return &nb.Bool
	}
	return nil
}

// FromNullFloat64 converts sql.NullFloat64 to *float64. Returns nil if not valid.
func FromNullFloat64(nf sql.NullFloat64) *float64 {
	if nf.Valid {
		return &nf.Float64
	}
	return nil
}

// FromNullRawMessage converts pqtype.NullRawMessage to map[string]interface{}. Returns nil if not valid or unmarshal fails.
func FromNullRawMessage(nrm pqtype.NullRawMessage) map[string]interface{} {
	if nrm.Valid && nrm.RawMessage != nil {
		var m map[string]interface{}
		if err := json.Unmarshal(nrm.RawMessage, &m); err == nil {
			return m
		}
	}
	return nil
}

// MapStructFields maps fields from a source struct to a destination struct,
// applying FromNull* conversions for sql.Null* and pqtype.NullRawMessage types.
// dst must be a pointer to a struct.
func MapStructFields(src interface{}, dst interface{}) error {
	srcVal := reflect.ValueOf(src)
	dstVal := reflect.ValueOf(dst)

	if dstVal.Kind() != reflect.Ptr || dstVal.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dst must be a pointer to a struct")
	}
	dstElem := dstVal.Elem()
	dstType := dstElem.Type()

	if srcVal.Kind() == reflect.Ptr {
		srcVal = srcVal.Elem()
	}
	if srcVal.Kind() != reflect.Struct {
		return fmt.Errorf("src must be a struct or a pointer to a struct")
	}

	for i := 0; i < dstType.NumField(); i++ {
		dstField := dstType.Field(i)
		dstFieldVal := dstElem.Field(i)

		// Skip unexported fields
		if !dstFieldVal.CanSet() {
			continue
		}

		// Find corresponding field in source struct by name
		srcFieldVal := srcVal.FieldByName(dstField.Name)
		if !srcFieldVal.IsValid() {
			continue // Source field not found
		}

		// Perform conversion based on source field type
		switch srcFieldVal.Type() {
		case reflect.TypeOf(sql.NullString{}):
			dstFieldVal.Set(reflect.ValueOf(FromNullString(srcFieldVal.Interface().(sql.NullString))))
		case reflect.TypeOf(sql.NullInt64{}):
			// Check if destination expects *int64 (for nullable int64) or int64 (for non-nullable int64)
			if dstFieldVal.Type() == reflect.TypeOf((*int64)(nil)) { // If destination is *int64
				dstFieldVal.Set(reflect.ValueOf(FromNullInt64Ptr(srcFieldVal.Interface().(sql.NullInt64))))
			} else if dstFieldVal.Type() == reflect.TypeOf(int64(0)) { // If destination is int64
				dstFieldVal.Set(reflect.ValueOf(FromNullInt64(srcFieldVal.Interface().(sql.NullInt64))))
			} else {
				// Fallback or error if type mismatch
				// For now, let's assume direct assignment if possible
				if srcFieldVal.Type().AssignableTo(dstFieldVal.Type()) {
					dstFieldVal.Set(srcFieldVal)
				} else if srcFieldVal.Type().ConvertibleTo(dstFieldVal.Type()) {
					dstFieldVal.Set(srcFieldVal.Convert(dstFieldVal.Type()))
				}
			}
		case reflect.TypeOf(sql.NullTime{}):
			dstFieldVal.Set(reflect.ValueOf(FromNullTime(srcFieldVal.Interface().(sql.NullTime))))
		case reflect.TypeOf(sql.NullBool{}):
			dstFieldVal.Set(reflect.ValueOf(FromNullBool(srcFieldVal.Interface().(sql.NullBool))))
		case reflect.TypeOf(sql.NullFloat64{}):
			dstFieldVal.Set(reflect.ValueOf(FromNullFloat64(srcFieldVal.Interface().(sql.NullFloat64))))
		case reflect.TypeOf(pqtype.NullRawMessage{}):
			dstFieldVal.Set(reflect.ValueOf(FromNullRawMessage(srcFieldVal.Interface().(pqtype.NullRawMessage))))
		default:
			// For non-nullable fields (like ID, CreatedAt, UpdatedAt, Title in db.Task)
			// and other direct types, attempt direct assignment if types are compatible.
			if srcFieldVal.Type().AssignableTo(dstFieldVal.Type()) {
				dstFieldVal.Set(srcFieldVal)
			} else if srcFieldVal.Type().ConvertibleTo(dstFieldVal.Type()) {
				dstFieldVal.Set(srcFieldVal.Convert(dstFieldVal.Type()))
			}
		}
	}
	return nil
}

// This is a dummy comment to force a file change and re-compilation. Let's see if any underlying errors surface.
