package store

import (
	"context"
	"time"
)

type Cluster struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	SecretName      string    `json:"secret_name"`
	SecretNamespace string    `json:"secret_namespace"`
	CreatedAt       time.Time `json:"created_at"`
	LastSeenAt      *time.Time `json:"last_seen_at,omitempty"`
	Reachable       bool      `json:"reachable"`
}

func (db *DB) ListClusters(ctx context.Context) ([]Cluster, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, name, secret_name, secret_namespace, created_at, last_seen_at, reachable
		FROM clusters ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clusters []Cluster
	for rows.Next() {
		var c Cluster
		if err := rows.Scan(&c.ID, &c.Name, &c.SecretName, &c.SecretNamespace, &c.CreatedAt, &c.LastSeenAt, &c.Reachable); err != nil {
			return nil, err
		}
		clusters = append(clusters, c)
	}
	return clusters, rows.Err()
}

func (db *DB) CreateCluster(ctx context.Context, name, secretName, secretNamespace string) (*Cluster, error) {
	var c Cluster
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO clusters (name, secret_name, secret_namespace)
		VALUES ($1, $2, $3)
		RETURNING id, name, secret_name, secret_namespace, created_at, last_seen_at, reachable`,
		name, secretName, secretNamespace,
	).Scan(&c.ID, &c.Name, &c.SecretName, &c.SecretNamespace, &c.CreatedAt, &c.LastSeenAt, &c.Reachable)
	return &c, err
}

func (db *DB) GetCluster(ctx context.Context, id string) (*Cluster, error) {
	var c Cluster
	err := db.Pool.QueryRow(ctx, `
		SELECT id, name, secret_name, secret_namespace, created_at, last_seen_at, reachable
		FROM clusters WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.SecretName, &c.SecretNamespace, &c.CreatedAt, &c.LastSeenAt, &c.Reachable)
	return &c, err
}

func (db *DB) DeleteCluster(ctx context.Context, id string) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM clusters WHERE id = $1`, id)
	return err
}

func (db *DB) UpdateClusterReachable(ctx context.Context, id string, reachable bool) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE clusters SET reachable = $1, last_seen_at = now() WHERE id = $2`, reachable, id)
	return err
}
