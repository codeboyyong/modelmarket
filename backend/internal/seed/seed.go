package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
			InputTokenPrice             float64 `json:"input_token_price"`
			InputTokenPriceUnit         string  `json:"input_token_price_unit"`
			OutputTokenPrice            float64 `json:"output_token_price"`
			OutputTokenPriceUnit        string  `json:"output_token_price_unit"`
			ProviderInputTokenCost      float64 `json:"provider_input_token_cost"`
			ProviderInputTokenCostUnit  string  `json:"provider_input_token_cost_unit"`
			ProviderOutputTokenCost     float64 `json:"provider_output_token_cost"`
			ProviderOutputTokenCostUnit string  `json:"provider_output_token_cost_unit"`
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
	raw, err := os.ReadFile(filepath.Join(dir, "dev_test_data.json"))
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

	if err := resetTx(ctx, tx); err != nil {
		return err
	}

	orgID := "org_" + data.Organization.Slug
	projectID := "project_" + data.Project.Slug
	walletID := "wallet_" + data.Project.Slug

	if _, err := tx.ExecContext(ctx, `insert into organizations(id, name, slug) values($1, $2, $3)`, orgID, data.Organization.Name, data.Organization.Slug); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into projects(id, organization_id, name, slug, environment) values($1, $2, $3, $4, $5)`, projectID, orgID, data.Project.Name, data.Project.Slug, data.Project.Environment); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into api_keys(id, project_id, name, prefix, key_hash, scopes, status) values($1, $2, $3, $4, $5, $6, 'active')`, "api_key_dev_test", projectID, "Dev test API key", "mk_dev_tes", "8f10d494a55d6262aded7008470955c7b6175fa6afc719e837f3634c0e4d19aa", "models:read,chat:create"); err != nil {
		return err
	}

	for _, user := range data.Users {
		userID := "user_" + stableToken(user.Email)
		if _, err := tx.ExecContext(ctx, `insert into users(id, email, name) values($1, $2, $3)`, userID, user.Email, user.Name); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `insert into memberships(id, user_id, organization_id, role) values($1, $2, $3, $4)`, "membership_"+stableToken(user.Email), userID, orgID, user.Role); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `insert into wallets(id, project_id, paid_credits, promotional_credits) values($1, $2, $3, $4)`, walletID, projectID, data.Wallet.PaidCredits, data.Wallet.PromotionalCredits); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into ledger_transactions(id, wallet_id, transaction_type, amount, credit_type, status, reason, idempotency_key) values($1, $2, 'grant', $3, 'promotional', 'posted', 'dev test credits', 'dev-test-credit-grant')`, "ledger_dev_test_credit_grant", walletID, data.Wallet.PromotionalCredits); err != nil {
		return err
	}

	for _, provider := range data.Providers {
		metadata, _ := json.Marshal(provider.Metadata)
		if _, err := tx.ExecContext(ctx, `insert into providers(id, slug, name, endpoint_url, metadata) values($1, $2, $3, $4, $5)`, "provider_"+provider.Slug, provider.Slug, provider.Name, provider.EndpointURL, string(metadata)); err != nil {
			return err
		}
	}

	for _, model := range data.Models {
		capabilities, _ := json.Marshal(model.Capabilities)
		params, _ := json.Marshal(model.Profile.DefaultParameters)
		providerID := "provider_" + model.ProviderSlug
		modelID := "model_" + model.Slug
		profileID := "profile_" + model.Profile.Slug
		if _, err := tx.ExecContext(ctx, `insert into models(id, provider_id, slug, name, modality, context_window, capabilities) values($1, $2, $3, $4, $5, $6, $7)`, modelID, providerID, model.Slug, model.Name, model.Modality, model.ContextWindow, string(capabilities)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `insert into model_versions(id, model_id, version) values($1, $2, 'dev')`, "model_version_"+model.Slug+"_dev", modelID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `insert into model_profiles(id, model_id, slug, name, system_prompt, default_parameters) values($1, $2, $3, $4, $5, $6)`, profileID, modelID, model.Profile.Slug, model.Profile.Name, model.Profile.SystemPrompt, string(params)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `insert into price_rules(id, model_id, model_profile_id, input_token_price, input_token_price_unit, output_token_price, output_token_price_unit, provider_input_token_cost, provider_input_token_cost_unit, provider_output_token_cost, provider_output_token_cost_unit) values($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`, "price_"+model.Profile.Slug, modelID, profileID, model.Price.InputTokenPrice, model.Price.InputTokenPriceUnit, model.Price.OutputTokenPrice, model.Price.OutputTokenPriceUnit, model.Price.ProviderInputTokenCost, model.Price.ProviderInputTokenCostUnit, model.Price.ProviderOutputTokenCost, model.Price.ProviderOutputTokenCostUnit); err != nil {
			return err
		}
	}

	if data.Conversation.Title != "" {
		conversationID := "conversation_seed_demo"
		if _, err := tx.ExecContext(ctx, `insert into conversations(id, project_id, title) values($1, $2, $3)`, conversationID, projectID, data.Conversation.Title); err != nil {
			return err
		}
		for index, message := range data.Conversation.Messages {
			if _, err := tx.ExecContext(ctx, `insert into messages(id, conversation_id, role, content) values($1, $2, $3, $4)`, fmt.Sprintf("message_seed_%d", index+1), conversationID, message.Role, message.Content); err != nil {
				return err
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `insert into usage_events(id, project_id, inference_request_id, model_slug, provider_slug, event_type, customer_charge, provider_cost, metadata) values($1, $2, null, $3, $4, $5, $6, $7, $8)`, "usage_dev_test_1", projectID, "mock-chat-default", "mock-provider", "dev_test_demo_usage", 1, 0, `{"source":"dev_test_data.json"}`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into routing_policies(id, project_id, name, status, policy) values($1, $2, $3, 'active', $4)`, "routing_policy_demo_default", projectID, "Demo default routing", `{"mode":"fixed","provider":"mock-provider"}`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into budget_policies(id, project_id, name, monthly_credit_limit, status, metadata) values($1, $2, $3, $4, 'active', '{}')`, "budget_policy_demo_monthly", projectID, "Demo monthly budget", 100000); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into coupons(id, code, status, credit_amount, metadata) values($1, $2, 'active', $3, $4)`, "coupon_demo_credits", "DEV-CREDITS", 1000, `{"source":"dev_test_data.json"}`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into payment_methods(id, organization_id, project_id, user_id, provider, provider_payment_method_id, method_type, display_name, last_four, exp_month, exp_year, billing_email, status, is_default, metadata) values($1, $2, $3, $4, $5, $6, 'card', $7, $8, $9, $10, $11, 'active', 1, $12)`, "payment_method_demo_card", orgID, projectID, "user_admin_example_com", "mock-payment", "pm_mock_demo_card", "Mock Visa ending 4242", "4242", 12, 2030, "admin@example.com", `{"source":"dev_test_data.json"}`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into payments(id, project_id, provider, provider_payment_id, status, amount_cents, currency, credits_granted, metadata) values($1, $2, 'mock-payment', $3, 'succeeded', $4, 'USD', $5, $6)`, "payment_demo_topup", projectID, "pay_mock_demo_topup", 10000, 10000, `{"source":"dev_test_data.json"}`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into payment_transactions(id, payment_id, payment_method_id, project_id, transaction_type, provider, provider_transaction_id, status, amount_cents, currency, credits_delta, idempotency_key, metadata) values($1, $2, $3, $4, 'top_up', 'mock-payment', $5, 'succeeded', $6, 'USD', $7, $8, $9)`, "payment_txn_demo_topup", "payment_demo_topup", "payment_method_demo_card", projectID, "txn_mock_demo_topup", 10000, 10000, "payment-txn-demo-topup", `{"source":"dev_test_data.json"}`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into user_interaction_history(id, user_id, organization_id, project_id, session_id, interaction_type, surface, target_type, target_id, event_name, metadata) values($1, $2, $3, $4, null, 'auth', 'dashboard', 'user', $2, 'dev_login', $5)`, "interaction_dev_test_login", "user_admin_example_com", orgID, projectID, `{"source":"dev_test_data.json"}`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into user_interaction_history(id, user_id, organization_id, project_id, session_id, interaction_type, surface, target_type, target_id, event_name, metadata) values($1, $2, $3, $4, null, 'view', 'catalog', 'model', $5, 'view_model_catalog', $6)`, "interaction_dev_test_catalog", "user_admin_example_com", orgID, projectID, "model_mock-chat", `{"source":"dev_test_data.json"}`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into user_interaction_history(id, user_id, organization_id, project_id, session_id, interaction_type, surface, target_type, target_id, event_name, metadata) values($1, $2, $3, $4, null, 'message', 'workbench', 'conversation', $5, 'send_chat_message', $6)`, "interaction_dev_test_workbench", "user_admin_example_com", orgID, projectID, "conversation_seed_demo", `{"source":"dev_test_data.json"}`); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	logger.Info("dev_test_data_loaded", "dir", dir)
	return nil
}

func Reset(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, deleteSeedSQL())
	if err != nil {
		return fmt.Errorf("reset dev test data: %w", err)
	}
	return nil
}

func resetTx(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, deleteSeedSQL())
	return err
}

func deleteSeedSQL() string {
	return `delete from audit_logs;
delete from user_interaction_history;
delete from provider_health_events;
delete from notifications;
delete from provider_settlements;
delete from webhook_deliveries;
delete from webhook_endpoints;
delete from budget_policies;
delete from routing_policies;
delete from coupons;
delete from invoice_items;
delete from invoices;
delete from payment_transactions;
delete from payment_methods;
delete from payments;
delete from usage_events;
delete from job_events;
delete from async_jobs;
delete from provider_attempts;
delete from inference_requests;
delete from message_attachments;
delete from workspace_assets;
delete from embedding_records;
delete from file_extractions;
delete from uploaded_files;
delete from context_summaries;
delete from conversation_branches;
delete from messages;
delete from conversations;
delete from ledger_transactions;
delete from wallets;
delete from price_rules;
delete from model_profiles;
delete from model_versions;
delete from models;
delete from providers;
delete from api_keys;
delete from sessions;
delete from oauth_accounts;
delete from memberships;
delete from projects;
delete from organizations;
delete from users;`
}

func stableToken(value string) string {
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, "@", "_")
	value = strings.ReplaceAll(value, ".", "_")
	value = strings.ReplaceAll(value, "-", "_")
	return value
}
