package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// MigrateNotificationDataProto converts legacy JSON text stored in data_proto BYTEA
// to proper proto-serialized google.protobuf.Struct bytes.
// Runs once at startup; idempotent — rows already in proto format are skipped.
func MigrateNotificationDataProto(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `SELECT id, data_proto FROM notifications WHERE data_proto IS NOT NULL`)
	if err != nil {
		return fmt.Errorf("migrate notification data: query: %w", err)
	}
	defer rows.Close()

	type updateRow struct {
		id   string
		data []byte
	}
	var updates []updateRow
	for rows.Next() {
		var r updateRow
		if err := rows.Scan(&r.id, &r.data); err != nil {
			return fmt.Errorf("migrate notification data: scan: %w", err)
		}
		updates = append(updates, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("migrate notification data: rows: %w", err)
	}

	for _, r := range updates {
		// Try proto unmarshal first — if it succeeds, already proto binary.
		var s structpb.Struct
		if proto.Unmarshal(r.data, &s) == nil {
			continue
		}
		// Try JSON unmarshal — legacy text stored as bytes.
		var m map[string]interface{}
		if json.Unmarshal(r.data, &m) != nil {
			continue
		}
		st, err := structpb.NewStruct(m)
		if err != nil {
			continue
		}
		out, err := proto.Marshal(st)
		if err != nil {
			continue
		}
		_, err = pool.Exec(ctx, `UPDATE notifications SET data_proto = $2 WHERE id = $1`, r.id, out)
		if err != nil {
			return fmt.Errorf("migrate notification data: update %s: %w", r.id, err)
		}
	}
	return nil
}
