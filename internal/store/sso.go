package store

import (
	"context"
	"time"
)

type SSOProvider struct {
	Provider     string    `json:"provider"`
	Enabled      bool      `json:"enabled"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	TenantID     string    `json:"tenant_id"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (db *DB) ListSSOProviders(ctx context.Context) ([]SSOProvider, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT provider, enabled, client_id, client_secret, tenant_id, updated_at
		FROM sso_providers ORDER BY provider`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SSOProvider
	for rows.Next() {
		var p SSOProvider
		if err := rows.Scan(&p.Provider, &p.Enabled, &p.ClientID, &p.ClientSecret, &p.TenantID, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (db *DB) GetSSOProvider(ctx context.Context, provider string) (SSOProvider, error) {
	var p SSOProvider
	err := db.Pool.QueryRow(ctx, `
		SELECT provider, enabled, client_id, client_secret, tenant_id, updated_at
		FROM sso_providers WHERE provider = $1`, provider).
		Scan(&p.Provider, &p.Enabled, &p.ClientID, &p.ClientSecret, &p.TenantID, &p.UpdatedAt)
	return p, err
}

func (db *DB) SetSSOProvider(ctx context.Context, p SSOProvider) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sso_providers (provider, enabled, client_id, client_secret, tenant_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (provider) DO UPDATE SET
			enabled       = EXCLUDED.enabled,
			client_id     = EXCLUDED.client_id,
			client_secret = EXCLUDED.client_secret,
			tenant_id     = EXCLUDED.tenant_id,
			updated_at    = now()`,
		p.Provider, p.Enabled, p.ClientID, p.ClientSecret, p.TenantID)
	return err
}
