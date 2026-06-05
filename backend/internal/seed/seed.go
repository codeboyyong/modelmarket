package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type Data struct {
	Users []struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	} `json:"users"`
	Organization struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"organization"`
	Project struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Environment string `json:"environment"`
	} `json:"project"`
	Wallet struct {
		PaidCredits        int64 `json:"paid_credits"`
		PromotionalCredits int64 `json:"promotional_credits"`
	} `json:"wallet"`
	Providers []struct {
		Slug        string         `json:"slug"`
		Name        string         `json:"name"`
		EndpointURL string         `json:"endpoint_url"`
		Metadata    map[string]any `json:"metadata"`
	} `json:"providers"`
	Models []struct {
		ProviderSlug  string         `json:"provider_slug"`
		Slug          string         `json:"slug"`
		Name          string         `json:"name"`
		Modality      string         `json:"modality"`
		ContextWindow int            `json:"context_window"`
		Capabilities  map[string]any `json:"capabilities"`
		Profile       struct {
			Slug              string         `json:"slug"`
			Name              string         `json:"name"`
			SystemPrompt      string         `json:"system_prompt"`
			DefaultParameters map[string]any `json:"default_parameters"`
		} `json:"profile"`
		Price struct {
			UnitType             string `json:"unit_type"`
			CustomerPriceCredits int    `json:"customer_price_credits"`
			ProviderCostCredits  int    `json:"provider_cost_credits"`
		} `json:"price"`
	} `json:"models"`
	Conversation struct {
		Title    string `json:"title"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	} `json:"conversation"`
}

func Load(ctx context.Context, db *sql.DB, dir string, logger *slog.Logger) error {
	raw, err := os.ReadFile(filepath.Join(dir, "dev_seed.json"))
	if err != nil {
		return err
	}
	var data Data
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var orgID string
	if err := tx.QueryRowContext(ctx, `
		insert into organizations(name, slug) values($1, $2)
		on conflict(slug) do update set name = excluded.name
		returning id`, data.Organization.Name, data.Organization.Slug).Scan(&orgID); err != nil {
		return err
	}

	var projectID string
	if err := tx.QueryRowContext(ctx, `
		insert into projects(organization_id, name, slug, environment) values($1, $2, $3, $4)
		on conflict(organization_id, slug) do update set name = excluded.name, environment = excluded.environment
		returning id`, orgID, data.Project.Name, data.Project.Slug, data.Project.Environment).Scan(&projectID); err != nil {
		return err
	}

	for _, user := range data.Users {
		var userID string
		if err := tx.QueryRowContext(ctx, `
			insert into users(email, name) values($1, $2)
			on conflict(email) do update set name = excluded.name
			returning id`, user.Email, user.Name).Scan(&userID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			insert into memberships(user_id, organization_id, role) values($1, $2, $3)
			on conflict(user_id, organization_id) do update set role = excluded.role`, userID, orgID, user.Role); err != nil {
			return err
		}
	}

	var walletID string
	if err := tx.QueryRowContext(ctx, `
		insert into wallets(project_id, paid_credits, promotional_credits) values($1, $2, $3)
		on conflict(project_id) do update set paid_credits = excluded.paid_credits, promotional_credits = excluded.promotional_credits, updated_at = now()
		returning id`, projectID, data.Wallet.PaidCredits, data.Wallet.PromotionalCredits).Scan(&walletID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		insert into ledger_transactions(wallet_id, transaction_type, amount, credit_type, status, reason, idempotency_key)
		values($1, 'grant', $2, 'promotional', 'posted', 'dev seed credits', 'dev-seed-credit-grant')
		on conflict(idempotency_key) do nothing`, walletID, data.Wallet.PromotionalCredits); err != nil {
		return err
	}

	for _, provider := range data.Providers {
		metadata, _ := json.Marshal(provider.Metadata)
		if _, err := tx.ExecContext(ctx, `
			insert into providers(slug, name, endpoint_url, metadata) values($1, $2, $3, $4::jsonb)
			on conflict(slug) do update set name = excluded.name, endpoint_url = excluded.endpoint_url, metadata = excluded.metadata`, provider.Slug, provider.Name, provider.EndpointURL, string(metadata)); err != nil {
			return err
		}
	}

	for _, model := range data.Models {
		capabilities, _ := json.Marshal(model.Capabilities)
		params, _ := json.Marshal(model.Profile.DefaultParameters)
		var providerID, modelID, profileID string
		if err := tx.QueryRowContext(ctx, `select id from providers where slug = $1`, model.ProviderSlug).Scan(&providerID); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `
			insert into models(provider_id, slug, name, modality, context_window, capabilities)
			values($1, $2, $3, $4, $5, $6::jsonb)
			on conflict(slug) do update set name = excluded.name, modality = excluded.modality, context_window = excluded.context_window, capabilities = excluded.capabilities
			returning id`, providerID, model.Slug, model.Name, model.Modality, model.ContextWindow, string(capabilities)).Scan(&modelID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			insert into model_versions(model_id, version) values($1, 'dev')
			on conflict(model_id, version) do nothing`, modelID); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `
			insert into model_profiles(model_id, slug, name, system_prompt, default_parameters)
			values($1, $2, $3, $4, $5::jsonb)
			on conflict(slug) do update set name = excluded.name, system_prompt = excluded.system_prompt, default_parameters = excluded.default_parameters, config_version = model_profiles.config_version + 1
			returning id`, modelID, model.Profile.Slug, model.Profile.Name, model.Profile.SystemPrompt, string(params)).Scan(&profileID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			insert into price_rules(model_id, model_profile_id, unit_type, customer_price_credits, provider_cost_credits)
			select $1, $2, $3, $4, $5
			where not exists(select 1 from price_rules where model_profile_id = $2 and unit_type = $3)`,
			modelID, profileID, model.Price.UnitType, model.Price.CustomerPriceCredits, model.Price.ProviderCostCredits); err != nil {
			return err
		}
	}

	if data.Conversation.Title != "" {
		var conversationID string
		if err := tx.QueryRowContext(ctx, `
			insert into conversations(project_id, title)
			select $1, $2
			where not exists(select 1 from conversations where project_id = $1 and title = $2)
			returning id`, projectID, data.Conversation.Title).Scan(&conversationID); err != nil {
			if err != sql.ErrNoRows {
				return err
			}
			if err := tx.QueryRowContext(ctx, `select id from conversations where project_id = $1 and title = $2`, projectID, data.Conversation.Title).Scan(&conversationID); err != nil {
				return err
			}
		}
		var messageCount int
		if err := tx.QueryRowContext(ctx, `select count(*) from messages where conversation_id = $1`, conversationID).Scan(&messageCount); err != nil {
			return err
		}
		if messageCount == 0 {
			for _, message := range data.Conversation.Messages {
				if _, err := tx.ExecContext(ctx, `insert into messages(conversation_id, role, content) values($1, $2, $3)`, conversationID, message.Role, message.Content); err != nil {
					return err
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	logger.Info("dev_seed_loaded", "dir", dir)
	return nil
}

func Reset(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `truncate audit_logs, usage_events, inference_requests, messages, conversations, ledger_transactions, wallets, price_rules, model_profiles, model_versions, models, providers, api_keys, sessions, oauth_accounts, memberships, projects, organizations, users restart identity cascade`)
	if err != nil {
		return fmt.Errorf("reset seed data: %w", err)
	}
	return nil
}
