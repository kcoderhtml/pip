package httpServer

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	shorturl "github.com/aviddiviner/shortcode-go"
	"github.com/kcoderhtml/pip/db"
	"github.com/uptrace/bun"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

type Service struct {
	*http.Server
	db *bun.DB
}

var (
	unsafeMarkdown = goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.Typographer,
			extension.GFM,
			emoji.Emoji,
			highlighting.NewHighlighting(
				// similar to the HTML rendering, we don't care about the style here
				highlighting.WithCustomStyle(styles.Fallback),
				highlighting.WithFormatOptions(
					html.WithClasses(true),
					html.WithAllClasses(true),
				),
			)),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			// this is ✨dangerous✨, but we're passing the output to bluemonday
			gmhtml.WithUnsafe(),
		),
	)
)

func NewServer(addr string, db *bun.DB) *Service {
	mux := http.NewServeMux()
	s := &Service{
		Server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
		db: db,
	}
	mux.HandleFunc("/", s.pasteHandler)
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// handle error
		}
	}()
	return s
}

func (s *Service) pasteHandler(w http.ResponseWriter, r *http.Request) {
	tmplPath := "httpServer/paste.tpl.html"
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Error parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	type Paste struct {
		Name     string
		Language string
		Content  string
		HTML     template.HTML
		Rendered bool
	}

	content, err := os.ReadFile("README.md")
	if err != nil {
		http.Error(w, "Error reading file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	paste := Paste{
		Name:     "README.md",
		Language: "markdown",
		Content:  string(content),
	}

	if r.URL.Path != "/" {
		dbPaste, err := db.GetPaste(s.db, shorturl.DecodeID(r.URL.Path[1:]))
		if err != nil {
			http.Error(w, "Error getting paste: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if strings.ToLower(dbPaste.Language) == "markdown" || strings.ToLower(dbPaste.Language) == "tex" {
			unsafeMarkdownHTML := bytes.NewBuffer(nil)

			if err := unsafeMarkdown.Convert([]byte(dbPaste.Content), unsafeMarkdownHTML); err != nil {
				http.Error(w, "Error converting markdown: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// sanitized := htmlSanitizer.Sanitize(unsafeMarkdownHTML.String())
			wrapped := fmt.Sprintf("<div class=\"markdown\">%s</div>", unsafeMarkdownHTML.String())

			paste.HTML = template.HTML(wrapped)
			paste.Rendered = true
		} else {
			paste.Content = dbPaste.Content
		}

		paste.Name = shorturl.EncodeID(int(dbPaste.ID))
		paste.Language = dbPaste.Language
	}

	if err := tmpl.Execute(w, paste); err != nil {
		http.Error(w, "Error executing template: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Service) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}
