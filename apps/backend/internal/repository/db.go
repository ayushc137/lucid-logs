package repository

import (
	"context"
	"fmt"

	"github.com/daily-journal/backend/internal/config"
	"github.com/surrealdb/surrealdb.go"
)

type DB struct {
	Client *surrealdb.DB
	ctx    context.Context
}

func NewDB(cfg config.DBConfig) (*DB, error) {
	addr := fmt.Sprintf("ws://%s:%s/rpc", cfg.Host, cfg.Port)

	client, err := surrealdb.New(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to surrealdb: %w", err)
	}

	ctx := context.Background()
	auth := &surrealdb.Auth{
		Username: cfg.User,
		Password: cfg.Pass,
	}

	if _, err := client.SignIn(ctx, auth); err != nil {
		return nil, fmt.Errorf("failed to sign in to surrealdb: %w", err)
	}

	if err := client.Use(ctx, cfg.Namespace, cfg.Database); err != nil {
		return nil, fmt.Errorf("failed to select namespace/database: %w", err)
	}

	return &DB{Client: client, ctx: ctx}, nil
}

func (d *DB) Close() {
	if err := d.Client.Close(d.Context()); err != nil {
		fmt.Printf("failed to close surrealdb connection: %v\n", err)
	}
}

func (d *DB) Context() context.Context {
	if d == nil || d.ctx == nil {
		return context.Background()
	}
	return d.ctx
}
