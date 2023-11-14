package text

import (
	"context"
)

func GetBool(ctx context.Context, val string) bool {
	if ret, ok := ctx.Value(val).(bool); ok {
		return ret
	}
	return false
}

func GetInt64(ctx context.Context, val string) int64 {
	if ret, ok := ctx.Value(val).(int64); ok {
		return ret
	}
	return 0
}

func GetString(ctx context.Context, val string) string {
	if ret, ok := ctx.Value(val).(string); ok {
		return ret
	}
	return ""
}

func GetFloat(ctx context.Context, val string) float64 {
	if ret, ok := ctx.Value(val).(float64); ok {
		return ret
	}
	return 0.0
}

func GetByte(ctx context.Context, val string) []byte {
	if ret, ok := ctx.Value(val).([]byte); ok {
		return ret
	}
	return []byte{}
}

func GetStrings(ctx context.Context, val string) []string {
	if ret, ok := ctx.Value(val).([]string); ok {
		return ret
	}
	return []string{}
}
