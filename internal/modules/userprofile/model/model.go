package model

import "oil/shared/model"

const (
	TableName  = "user_profiles"
	EntityName = "user_profile"

	FieldID         = "id"
	FieldAuthUserID = "auth_user_id"
	FieldEmail      = "email"
	FieldRole       = "role"
	FieldName       = "name"
	FieldImage      = "image"
	FieldActive     = "active"
)

type Userprofile struct {
	ID         string `db:"id"`
	AuthUserID string `db:"auth_user_id"`
	Email      string `db:"email"`
	Role       string `db:"role"`
	Name       string `db:"name"`
	Image      string `db:"image"`
	Active     bool   `db:"active"`
	model.Metadata
}
