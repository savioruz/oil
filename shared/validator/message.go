package validator

var (
	messages = map[string]string{
		"required": "{field} is required",
		"gte":      "{field} must be greater than or equal to {param}",
		"lte":      "{field} must be less than or equal to {param}",
		"oneof":    "{field} must be one of {param}",
		"max":      "{field} must be less than or equal to {param}",
		"min":      "{field} must be greater than or equal to {param}",
		"email":    "{field} must be a valid email address",
		"url":      "{field} must be a valid URL",
		"dive":     "{field} contains a forbidden value",
		"mimetype": "{field} must be one of the allowed file types: {param}",
	}
)
