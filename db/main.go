package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/uptrace/bun"

	gossh "golang.org/x/crypto/ssh"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID      int64    `bun:"id,pk,autoincrement"`
	Name    string   `bun:"name,notnull,unique"`
	SshKeys []SshKey `bun:"ssh_keys,type:jsonb"`
}

type SshKey struct {
	Fingerprint string `bun:"user_id"`
	Type        string `bun:"key,notnull"`
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

// get or create user
func GetUser(db *bun.DB, sess ssh.Session) (*User, string, error) {
	if sess.User() == "" {
		return nil, "somehow you managed to break ssh?", errors.New("no user")
	}
	user := &User{}
	err := db.NewSelect().Model(user).Where("name = ?", sess.User()).Scan(context.Background())

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, "there was an error finding you in the db", err
	}

	fingerprint := gossh.FingerprintSHA256(sess.PublicKey())

	if user.Name == "" {
		user.Name = sess.User()
		user.SshKeys = []SshKey{
			{
				Type:        sess.PublicKey().Type(),
				Fingerprint: fingerprint,
			},
		}

		_, err := db.NewInsert().Model(user).Exec(context.Background())
		if err != nil {
			return nil, "there was an error creating you in the db", err
		}

		log.Info("Created new user", "name", user.Name, "keyType", user.SshKeys[0].Type)

		return user, "welcome to pip %s!", nil
	}

	// check whether the user's key is authorized
	for _, key := range user.SshKeys {
		if key.Fingerprint == fingerprint && key.Type == sess.PublicKey().Type() {
			log.Info("Authorized user", "name", user.Name, "keyType", key.Type)
			return user, "welcome back to pip %s!", nil
		}
	}

	log.Warn("Unauthorized user", "name", user.Name, "keyType", sess.PublicKey().Type())

	return user, "your key doesn't match; try another?", errors.New("unauthorized")
}
