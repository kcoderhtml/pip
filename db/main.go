package db

import (
	"context"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID      int64    `bun:"id,pk,autoincrement"`
	Name    string   `bun:"name,notnull,unique"`
	SshKeys []string `bun:"ssh_keys,type:jsonb"`
}

// create tables if they don't exist
func CreateSchema(db *bun.DB) error {
	models := []interface{}{
		(*User)(nil),
	}

	for _, model := range models {
		if _, err := db.NewCreateTable().Model(model).IfNotExists().Exec(context.Background()); err != nil {
			return err
		}
	}

	return nil
}
