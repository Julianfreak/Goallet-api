package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host    string
	Port    string
	User    string
	DBName  string
	SSLMode string
	Region  string
	Creds   aws.CredentialsProvider
}

func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s@%s:%s/%s?sslmode=%s",
		strings.TrimSpace(cfg.User),
		strings.TrimSpace(cfg.Host),
		strings.TrimSpace(cfg.Port),
		strings.TrimSpace(cfg.DBName),
		strings.TrimSpace(cfg.SSLMode),
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("error parseando config de pgxpool: %w", err)
	}

	endpoint := fmt.Sprintf("%s:%s", strings.TrimSpace(cfg.Host), strings.TrimSpace(cfg.Port))
	region := strings.TrimSpace(cfg.Region)
	dbUser := strings.TrimSpace(cfg.User)

	poolCfg.BeforeConnect = func(connectCtx context.Context, cc *pgx.ConnConfig) error {
		token, err := auth.BuildAuthToken(connectCtx, endpoint, region, dbUser, cfg.Creds)
		if err != nil {
			return fmt.Errorf("error generando token IAM: %w", err)
		}
		cc.Password = token
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("error creando pgxpool: %w", err)
	}

	return pool, nil
}
