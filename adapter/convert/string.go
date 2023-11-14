package convert

import (
	"fmt"
	"strconv"
)

// GetString converts interface to string.
func GetString(v interface{}) string {
	switch result := v.(type) {
	case float64:
		return strconv.FormatFloat(result, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(result, 10)
	case string:
		return result
	case int32:
		return strconv.FormatInt(int64(result), 10)
	case float32:
		return strconv.FormatFloat(float64(result), 'f', -1, 32)
	case int:
		return strconv.FormatInt(int64(result), 10)
	case []byte:
		return string(result)
	default:
		if v != nil {
			return fmt.Sprint(result)
		}
	}
	return ""
}

// GetInt converts interface to int.
func GetInt(v interface{}) int {
	switch result := v.(type) {
	case int:
		return result
	case int32:
		return int(result)
	case int64:
		return int(result)
	case float32:
		return int(result)
	case float64:
		return int(result)
	default:
		if d := GetString(v); d != "" {
			value, _ := strconv.Atoi(d)
			return value
		}
	}
	return 0
}

// GetInt64 converts interface to int64.
func GetInt64(v interface{}) int64 {
	switch result := v.(type) {
	case string:
		value, err := strconv.ParseInt(v.(string), 10, 64)
		if err != nil {
			return 0
		}
		return value
	case int32:
		return int64(result)
	case int:
		return int64(result)
	case int64:
		return result
	default:
		if d := GetString(v); d != "" {
			value, _ := strconv.ParseInt(d, 10, 64)
			return value
		}
	}
	return 0
}

// GetFloat64 converts interface to float64.
func GetFloat64(v interface{}) float64 {
	switch result := v.(type) {
	case string:
		//i := strings.Index(v.(string), ".")
		//l := len(v.(string)) - 1
		//f := 0
		//if i >= 0 {
		//	f = l - i
		//}
		value, err := strconv.ParseFloat(v.(string), 64)
		if err != nil {
			return 0.0
		}
		return value
	case int64:
		return float64(result)
	case int:
		return float64(result)
	case float32:
		return float64(result)
	case int32:
		return float64(result)
	case float64:
		return result
	default:
		if d := GetString(v); d != "" {
			value, _ := strconv.ParseFloat(d, 64)
			return value
		}
	}
	return 0
}

// GetBool converts interface to bool.
func GetBool(v interface{}) bool {
	switch result := v.(type) {
	case bool:
		return result
	default:
		if d := GetString(v); d != "" {
			value, _ := strconv.ParseBool(d)
			return value
		}
	}
	return false
}
