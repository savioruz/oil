package model

import "time"

type Metadata struct {
	CreatedAt  time.Time `db:"created_at"`
	ModifiedAt time.Time `db:"modified_at"`
	CreatedBy  string    `db:"created_by"`
	ModifiedBy string    `db:"modified_by"`
}
