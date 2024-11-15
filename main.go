package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
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

	shorturl "github.com/aviddiviner/shortcode-go"
	humanize "github.com/dustin/go-humanize"

	database "github.com/kcoderhtml/pip/db"
	"github.com/kcoderhtml/pip/styles"
	"github.com/kcoderhtml/pip/utils"
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

	detector, err := utils.NewLangDetector(os.Getenv("GUESSLANG_URL"))
	if err != nil {
		log.Error("Could not initialize language detector", "error", err)
		return
	}
	log.Info("Initialized language detector", "url", os.Getenv("GUESSLANG_URL"))

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
						if !errors.Is(err, database.ErrUnauthorized) {
							log.Error("Could not get user", "error", err)
						}
						next(sess)
						return
					}

					wish.Println(sess, message)

					_, _, isPty := sess.Pty()
					if isPty {
						wish.Println(sess, styles.Normal.Render("\nTo upload a paste simply run: "+styles.Code.Render(" cat example.md | ssh dunkirk.sh")+"\n "))
						next(sess)
						return
					}

					// read any input
					content := make([]byte, 0)
					size := uint64(0)
					buf := make([]byte, 512*1024) // 512kib
					n, err := sess.Read(buf)
					isEOF := errors.Is(err, io.EOF)
					if err != nil && !isEOF {
						log.Error("Could not read from session", "error", err)
						return
					}

					size += uint64(n)
					content = append(content, buf[:n]...)

					fmt.Println("size", size, "n", n, "isEOF", isEOF, "err", err, "content", string(content))

					answer, err := detector.GetLang(string(buf))
					if err != nil {
						log.Error("Could not guess language", "error", err)
					}

					wish.Println(sess, styles.Normal.Render("\nDetected language: "+styles.Info.Render(answer)+"\nSize: "+styles.Info.Render(humanize.Bytes(size))+"\n"))

					res, err := database.CreatePaste(db, user, string(content), answer, "never")
					if err != nil {
						log.Error("Could not create paste", "error", err)
					}

					wish.Println(sess, styles.Info.Render("Paste Saved!"+"\n"+styles.Normal.Render("To view your paste visit: ")+styles.Code.Render("https://pip.dunkirk.sh/"+shorturl.EncodeID(int(res.ID)))))

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
