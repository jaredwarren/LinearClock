package server

import (
	"html/template"
	"testing"
	"time"
)

func TestTemplateParseSmoke(t *testing.T) {
	t.Parallel()

	files := []string{
		"../../templates/layout.html",
		"../../templates/home.html",
		"../../templates/events.html",
		"../../templates/test.html",
	}

	tmpl, err := template.New("base").Funcs(template.FuncMap{
		"ColorString":     ColorString,
		"TimeNum":         TimeNum,
		"TimeFormat":      TimeFormat,
		"TimeFormatLocal": TimeFormatLocal,
		"timefmt": func(tt time.Time) string {
			return tt.Format("15:04")
		},
		"add": func(i, j int) int { return i + j },
	}).ParseFiles(files...)
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	if tmpl.Lookup("base") == nil {
		t.Fatal("expected base template")
	}
}

