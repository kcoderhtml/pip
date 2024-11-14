package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/logging"
	"github.com/go-pg/pg/v10"
	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	database "github.com/kcoderhtml/pip/db"
)

const (
	host = "localhost"
	port = "23234"
)

func main() {
	err := godotenv.Load() // 👈 load .env file
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Connect to a database
	dsn := os.Getenv("DATABASE_URL")
	opt, err := pg.ParseURL(dsn)
	if err != nil {
		log.Error("Could not parse DSN", "error", err)
		return
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	// ping db
	err = db.Ping()
	if err != nil {
		log.Error("Could not connect to the database", "error", err)
		return
	}
	log.Info("Connected to the database", "addr", opt.Addr, "user", opt.User, "database", opt.Database)

	err = database.CreateSchema(db)

	if err != nil {
		log.Error("Could not create schema", "error", err)
		return
	}

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			return true
		}),
		wish.WithMiddleware(
			func(next ssh.Handler) ssh.Handler {
				return func(sess ssh.Session) {
					// if the current session's user public key is one of the
					// known users, we greet them and return.
					user, message, err := database.GetUser(db, sess)

					if err != nil {
						wish.Println(sess, message)
						log.Error("Could not get user", "error", err)
						next(sess)
						return
					}

					wish.Println(sess, fmt.Sprintf(message, user.Name))
					next(sess)
				}
			},
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	log.Info("Ready!")

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}
