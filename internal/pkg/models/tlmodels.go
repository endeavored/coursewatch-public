package models

import (
	"context"
)

type App struct {
	Ctx      context.Context
	Channels map[string]string
	Classes  map[string]bool
	Term     string
	Webhooks []string
}
