// Package i18n provides internationalization utility tests
// Author: Done-0
// Created: 2026-01-31
package i18n

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"

	"magnet2video/internal/types/consts"
)

func TestT_NoLocalizer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	got := T(c, "hello")
	if got != "hello" {
		t.Fatalf("T() = %q, want key", got)
	}
}

func TestT_WrongLocalizerType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(consts.LocalizerContextKey, "not-a-localizer")

	got := T(c, "hello")
	if got != "hello" {
		t.Fatalf("T() = %q, want key", got)
	}
}

func TestT_LocalizeWithParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	bundle := i18n.NewBundle(language.English)
	bundle.AddMessages(language.English, &i18n.Message{
		ID:    "hello",
		Other: "Hello {{.Name}}",
	})
	localizer := i18n.NewLocalizer(bundle, language.English.String())
	c.Set(consts.LocalizerContextKey, localizer)

	got := T(c, "hello", "Name", "Done")
	if got != "Hello Done" {
		t.Fatalf("T() = %q, want %q", got, "Hello Done")
	}
}

func TestT_MissingMessageReturnsKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	bundle := i18n.NewBundle(language.English)
	localizer := i18n.NewLocalizer(bundle, language.English.String())
	c.Set(consts.LocalizerContextKey, localizer)

	got := T(c, "missing")
	if got != "missing" {
		t.Fatalf("T() = %q, want key", got)
	}
}
