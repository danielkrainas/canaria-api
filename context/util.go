package context

func GetStringValue(ctx Context, key interface{}) string {
	if value, ok := ctx.Value(key).(string); ok {
		return value
	}

	return ""
}
