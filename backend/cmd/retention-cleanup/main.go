package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"model-market/backend/internal/config"
)

type retentionPolicy struct {
	ConversationDays int `json:"conversation_days"`
	AssetDays        int `json:"asset_days"`
}

func main() {
	apply := flag.Bool("apply", false, "delete expired rows and local objects; default is dry-run")
	flag.Parse()
	cfg := config.Load()
	db, err := sql.Open(cfg.SQLDriverName(), cfg.DatabaseURL)
	if err != nil {
		fail(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	rows, err := db.QueryContext(ctx, `select id, retention_policy from user_projects order by id`)
	if err != nil {
		fail(err)
	}
	defer rows.Close()
	totalConversations, totalAssets := 0, 0
	for rows.Next() {
		var projectID, raw string
		if err := rows.Scan(&projectID, &raw); err != nil {
			fail(err)
		}
		policy := retentionPolicy{ConversationDays: 365, AssetDays: 365}
		_ = json.Unmarshal([]byte(raw), &policy)
		conversations, assets, err := expiredIDs(ctx, db, projectID, policy)
		if err != nil {
			fail(err)
		}
		fmt.Printf("project=%s expired_conversations=%d expired_assets=%d\n", projectID, len(conversations), len(assets))
		totalConversations += len(conversations)
		totalAssets += len(assets)
		if *apply {
			for _, id := range conversations {
				if err := deleteConversation(ctx, db, id, cfg.ObjectStorageDir); err != nil {
					fail(err)
				}
			}
			for _, id := range assets {
				if err := deleteAsset(ctx, db, id, cfg.ObjectStorageDir); err != nil {
					fail(err)
				}
			}
		}
	}
	mode := "dry-run"
	if *apply {
		mode = "applied"
	}
	fmt.Printf("mode=%s conversations=%d assets=%d\n", mode, totalConversations, totalAssets)
}

func expiredIDs(ctx context.Context, db *sql.DB, projectID string, policy retentionPolicy) ([]string, []string, error) {
	conversations := []string{}
	assets := []string{}
	if policy.ConversationDays > 0 {
		rows, err := db.QueryContext(ctx, `select id from user_conversations where project_id = $1 and updated_at < $2`, projectID, time.Now().AddDate(0, 0, -policy.ConversationDays))
		if err != nil {
			return nil, nil, err
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, nil, err
			}
			conversations = append(conversations, id)
		}
		rows.Close()
	}
	if policy.AssetDays > 0 {
		rows, err := db.QueryContext(ctx, `select id from user_workbench_assets where project_id = $1 and conversation_id is null and created_at < $2`, projectID, time.Now().AddDate(0, 0, -policy.AssetDays))
		if err != nil {
			return nil, nil, err
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, nil, err
			}
			assets = append(assets, id)
		}
		rows.Close()
	}
	return conversations, assets, nil
}

func deleteConversation(ctx context.Context, db *sql.DB, id, storageRoot string) error {
	rows, err := db.QueryContext(ctx, `select object_key from user_workbench_assets where conversation_id = $1 and coalesce(object_key, '') <> ''`, id)
	if err != nil {
		return err
	}
	keys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return err
		}
		keys = append(keys, key)
	}
	rows.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, query := range []string{`delete from user_message_attachments where message_id in (select id from user_messages where conversation_id=$1) or asset_id in (select id from user_workbench_assets where conversation_id=$1)`, `delete from user_file_extractions where asset_id in (select id from user_workbench_assets where conversation_id=$1)`, `delete from user_workbench_assets where conversation_id=$1`, `delete from user_messages where conversation_id=$1`, `delete from user_conversation_branches where conversation_id=$1`, `delete from user_conversations where id=$1`} {
		if _, err := tx.ExecContext(ctx, query, id); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for _, key := range keys {
		removeObject(storageRoot, key)
	}
	return nil
}

func deleteAsset(ctx context.Context, db *sql.DB, id, storageRoot string) error {
	var key string
	if err := db.QueryRowContext(ctx, `select coalesce(object_key, '') from user_workbench_assets where id=$1`, id).Scan(&key); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, query := range []string{`delete from user_message_attachments where asset_id=$1`, `delete from user_file_extractions where asset_id=$1`, `delete from user_workbench_assets where id=$1`} {
		if _, err := tx.ExecContext(ctx, query, id); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	removeObject(storageRoot, key)
	return nil
}

func removeObject(root, key string) {
	clean := filepath.Clean(strings.Trim(strings.ReplaceAll(key, "\\", "/"), "/"))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(clean) {
		return
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return
	}
	path, err := filepath.Abs(filepath.Join(absRoot, clean))
	if err != nil || !strings.HasPrefix(path, absRoot+string(os.PathSeparator)) {
		return
	}
	_ = os.Remove(path)
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
