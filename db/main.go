package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/kcoderhtml/pip/styles"
	"github.com/uptrace/bun"

	gossh "golang.org/x/crypto/ssh"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID      int64    `bun:"id,pk,autoincrement"`
	Name    string   `bun:"name,notnull,unique"`
	SshKeys []SshKey `bun:"ssh_keys,type:jsonb"`
	Pastes  []string `bun:"pastes,type:jsonb"`
}

type Paste struct {
	bun.BaseModel `bun:"table:pastes,alias:p"`
	ID            int64  `bun:"id,pk,autoincrement"`
	Content       string `bun:"content,notnull"`
	Language      string `bun:"language,notnull"`
	Expiry        string `bun:"expiry,notnull"`
}

type SshKey struct {
	Fingerprint string `bun:"user_id"`
	Type        string `bun:"key,notnull"`
}

var (
	ErrUnauthorized = errors.New("unauthorized")
)

// create tables if they don't exist
func CreateSchema(db *bun.DB) error {
	models := []interface{}{
		(*User)(nil),
		(*Paste)(nil),
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
		return nil, styles.Error.Render("X there was an error finding you in the db"), err
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
			return nil, styles.Error.Render("X there was an error creating you in the db"), err
		}

		log.Info("Created new user", "name", user.Name, "keyType", user.SshKeys[0].Type)

		return user, styles.Info.Render(fmt.Sprintf("welcome to pip %s!", sess.User())), nil
	}

	// check whether the user's key is authorized
	for _, key := range user.SshKeys {
		if key.Fingerprint == fingerprint && key.Type == sess.PublicKey().Type() {
			log.Info("Authorized user", "name", user.Name, "keyType", key.Type)
			return user, styles.Info.Render(fmt.Sprintf("✔ welcome back to pip %s!", sess.User())), nil
		}
	}

	log.Warn("Unauthorized user", "name", user.Name, "keyType", sess.PublicKey().Type())

	return user, styles.Warn.Render("🔒your key doesn't match. try another?"), ErrUnauthorized
}

// create paste
func CreatePaste(db *bun.DB, user *User, content string, lang string, expiry string) (*Paste, error) {
	paste := &Paste{
		Content:  content,
		Language: lang,
		Expiry:   expiry,
	}

	_, err := db.NewInsert().Model(paste).Exec(context.Background())
	if err != nil {
		return nil, err
	}

	// add paste to user via sql
	user.Pastes = append(user.Pastes, paste.Content)
	_, err = db.NewUpdate().Model(user).Where("name = ?", user.Name).Set("pastes = ?", user.Pastes).Exec(context.Background())
	if err != nil {
		return nil, err
	}

	return paste, nil
}

func GetPaste(db *bun.DB, id int) (*Paste, error) {
	paste := &Paste{}
	err := db.NewSelect().Model(paste).Where("id = ?", id).Scan(context.Background())
	if err != nil {
		return nil, err
	}

	return paste, nil
}
