package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type response map[string]any

func (a *App) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, response{"status": "ok", "service": "model-market-backend"})
}

func (a *App) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.DB.PingContext(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, response{"status": "not_ready", "database": err.Error()})
		return
	}
	if a.Redis != nil {
		if err := a.Redis.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, response{"status": "not_ready", "redis": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, response{"status": "ready"})
}

func (a *App) devSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	counts := map[string]int64{}
	summaryTables := map[string]string{
		"users":          "sys_users",
		"organizations":  "sys_organizations",
		"projects":       "user_projects",
		"providers":      "sys_providers",
		"models":         "sys_models",
		"model_profiles": "sys_model_profiles",
		"wallets":        "user_wallets",
		"conversations":  "user_conversations",
		"messages":       "user_messages",
		"usage_events":   "user_usage_events",
	}
	for key, table := range summaryTables {
		var count int64
		if err := a.DB.QueryRowContext(ctx, fmt.Sprintf("select count(*) from %s", table)).Scan(&count); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		counts[key] = count
	}
	writeJSON(w, http.StatusOK, response{"dev_mode": a.Config.DevMode, "counts": counts})
}

func (a *App) devLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	if req.Email == "" {
		req.Email = "admin@example.com"
	}
	login, err := a.loginByEmail(r.Context(), req.Email)
	if err != nil {
		if a.Config.DevMode {
			writeJSON(w, http.StatusOK, devAuthResponse(req.Email, "Admin User"))
			return
		}
		writeJSON(w, http.StatusUnauthorized, response{"error": "dev_user_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, login)
}

func (a *App) passwordLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_credentials"})
		return
	}
	email, storedHash, err := a.lookupLoginIdentity(r.Context(), req.Username)
	if err != nil {
		if a.Config.DevMode {
			email := req.Username
			name := strings.Split(req.Username, "@")[0]
			if !strings.Contains(email, "@") {
				email = "admin@example.com"
				name = req.Username
			}
			writeJSON(w, http.StatusOK, devAuthResponse(email, name))
			return
		}
		writeJSON(w, http.StatusUnauthorized, response{"error": "invalid_credentials"})
		return
	}
	if storedHash != "" && storedHash != hashPassword(req.Password) {
		writeJSON(w, http.StatusUnauthorized, response{"error": "invalid_credentials"})
		return
	}
	login, err := a.loginByEmail(r.Context(), email)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, response{"error": "invalid_credentials"})
		return
	}
	writeJSON(w, http.StatusOK, login)
}

func (a *App) changePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_credentials"})
		return
	}
	email, _, err := a.lookupLoginIdentity(r.Context(), req.Username)
	if err != nil {
		writeJSON(w, http.StatusNotFound, response{"error": "user_not_found"})
		return
	}
	result, err := a.DB.ExecContext(r.Context(), `
		update sys_users
		set password_hash = $1
		where lower(email) = lower($2)`, hashPassword(req.Password), email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeJSON(w, http.StatusNotFound, response{"error": "user_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, response{"status": "password_updated", "email": email})
}

func (a *App) devSocialLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider != "google" && provider != "github" && provider != "facebook" {
		writeJSON(w, http.StatusBadRequest, response{"error": "unsupported_provider"})
		return
	}
	email := map[string]string{
		"google":   "admin@example.com",
		"github":   "admin@example.com",
		"facebook": "developer@example.com",
	}[provider]
	login, err := a.loginByEmail(r.Context(), email)
	if err != nil {
		if a.Config.DevMode {
			login = devAuthResponse(email, map[string]string{
				"google":   "Admin User",
				"github":   "Admin User",
				"facebook": "Developer User",
			}[provider])
			login["provider"] = provider
			writeJSON(w, http.StatusOK, login)
			return
		}
		writeJSON(w, http.StatusUnauthorized, response{"error": "dev_user_not_found"})
		return
	}
	login["provider"] = provider
	writeJSON(w, http.StatusOK, login)
}

func (a *App) signup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Name        string `json:"name"`
		Password    string `json:"password"`
		AccountType string `json:"account_type"`
		CompanyName string `json:"company_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	req.AccountType = strings.ToLower(strings.TrimSpace(req.AccountType))
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_signup_fields"})
		return
	}
	if req.Name == "" {
		req.Name = strings.Split(req.Email, "@")[0]
	}
	userType := "individual_consumer"
	orgID := "org-demo"
	var companyID *string
	if req.AccountType == "corporate" || req.AccountType == "corporate_member" {
		if req.CompanyName == "" {
			writeJSON(w, http.StatusBadRequest, response{"error": "missing_company_name"})
			return
		}
		var foundCompanyID, foundOrgID string
		err := a.DB.QueryRowContext(r.Context(), `
			select c.id, p.organization_id
			from user_companies c
			join user_projects p on p.company_id = c.id
			where lower(c.name) = lower($1)
			order by p.created_at asc
			limit 1`, req.CompanyName).Scan(&foundCompanyID, &foundOrgID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, response{"error": "company_not_found"})
			return
		}
		userType = "corporate_member"
		companyID = &foundCompanyID
		orgID = foundOrgID
	}

	userID := "user_" + randomHex(8)
	membershipID := "membership_" + randomHex(8)
	_, err := a.DB.ExecContext(r.Context(), `
		insert into sys_users(id, email, name, avatar_url, status, password_hash, user_type, company_id, ui_theme, language)
		values($1, $2, $3, null, 'active', $4, $5, $6, 'Light', 'EN')`, userID, req.Email, req.Name, hashPassword(req.Password), userType, companyID)
	if err != nil {
		if a.Config.DevMode {
			writeJSON(w, http.StatusCreated, devAuthResponse(req.Email, req.Name))
			return
		}
		writeJSON(w, http.StatusConflict, response{"error": "user_exists_or_invalid"})
		return
	}
	_, err = a.DB.ExecContext(r.Context(), `
		insert into sys_memberships(id, user_id, organization_id, role)
		values($1, $2, $3, 'developer')`, membershipID, userID, orgID)
	if err != nil {
		if a.Config.DevMode {
			writeJSON(w, http.StatusCreated, devAuthResponse(req.Email, req.Name))
			return
		}
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	login, err := a.loginByEmail(r.Context(), req.Email)
	if err != nil {
		if a.Config.DevMode {
			writeJSON(w, http.StatusCreated, devAuthResponse(req.Email, req.Name))
			return
		}
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, login)
}

func (a *App) lookupLoginIdentity(ctx context.Context, identity string) (string, string, error) {
	var email string
	var passwordHash sql.NullString
	err := a.DB.QueryRowContext(ctx, `
		select email, coalesce(password_hash, '')
		from sys_users
		where lower(email) = lower($1) or lower(name) = lower($1)
		limit 1`, identity).Scan(&email, &passwordHash)
	if err != nil {
		return "", "", err
	}
	return email, passwordHash.String, nil
}

func (a *App) loginByEmail(ctx context.Context, email string) (response, error) {
	var userID, orgID, projectID, name, userType string
	var companyID, companyName sql.NullString
	err := a.DB.QueryRowContext(ctx, `
		select u.id, u.name, u.user_type, coalesce(u.company_id, ''), coalesce(c.name, ''), o.id, p.id
		from sys_users u
		join sys_memberships m on m.user_id = u.id
		join sys_organizations o on o.id = m.organization_id
		join user_projects p on p.organization_id = o.id
		left join user_companies c on c.id = u.company_id
		where u.email = $1
		order by p.created_at asc
		limit 1`, email).Scan(&userID, &name, &userType, &companyID, &companyName, &orgID, &projectID)
	if err != nil {
		return nil, err
	}
	return response{
		"user":            authUserResponse(userID, email, name, userType, companyID.String, companyName.String),
		"organization_id": orgID,
		"project_id":      projectID,
		"session":         response{"access_token": "dev_" + randomHex(16), "token_type": "Bearer"},
	}, nil
}

func devAuthResponse(email, name string) response {
	if strings.TrimSpace(name) == "" {
		name = "Dev User"
	}
	userID := "user-dev-fallback"
	userType := "individual_consumer"
	if strings.EqualFold(email, "admin@example.com") {
		userID = "user-admin"
		userType = "sys_admin"
	}
	return response{
		"user":            authUserResponse(userID, email, name, userType, "", ""),
		"organization_id": "org-demo",
		"project_id":      "project-demo",
		"session":         response{"access_token": "dev_" + randomHex(16), "token_type": "Bearer"},
		"dev_fallback":    true,
	}
}

func authUserResponse(id, email, name, userType, companyID, companyName string) response {
	return response{"id": id, "email": email, "name": name, "user_type": userType, "company_id": companyID, "company_name": companyName}
}

func (a *App) models(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.QueryContext(r.Context(), `
		select m.id, m.slug, m.name, p.name, m.modality, m.status, coalesce(mp.slug, ''), coalesce(mp.name, ''), coalesce(mp.default_parameters, '{}')
		from sys_models m
		join sys_providers p on p.id = m.provider_id
		left join sys_model_profiles mp on mp.model_id = m.id and mp.status = 'public'
		order by p.name, m.name`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, slug, name, provider, modality, status, profileSlug, profileName, defaultParameters string
		if err := rows.Scan(&id, &slug, &name, &provider, &modality, &status, &profileSlug, &profileName, &defaultParameters); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{"id": id, "slug": slug, "name": name, "provider": provider, "modality": modality, "status": status, "profile_slug": profileSlug, "profile_name": profileName, "default_parameters": defaultParameters})
	}
	writeJSON(w, http.StatusOK, response{"models": items})
}

func (a *App) pricing(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.QueryContext(r.Context(), `
		select p.name, m.name, m.slug, m.modality, coalesce(mp.name, ''), coalesce(mp.slug, ''),
			pr.price_seq_id, pr.pricing_variant, pr.price_type, pr.price_unit, pr.price,
			pr.currency, pr.provider_price, pr.provider_currency, pr.price_metadata, pr.status
		from sys_price_rules pr
		join sys_models m on m.id = pr.model_id
		join sys_providers p on p.id = m.provider_id
		left join sys_model_profiles mp on mp.id = pr.model_profile_id
		order by p.name, m.modality, m.name, pr.price_seq_id, pr.price_type`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var provider, modelName, modelSlug, modality, profileName, profileSlug, pricingVariant, priceType, priceUnit, currency, providerCurrency, priceMetadata, status string
		var priceSeqID int
		var price, providerPrice float64
		if err := rows.Scan(&provider, &modelName, &modelSlug, &modality, &profileName, &profileSlug, &priceSeqID, &pricingVariant, &priceType, &priceUnit, &price, &currency, &providerPrice, &providerCurrency, &priceMetadata, &status); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{
			"provider":          provider,
			"model":             modelName,
			"model_slug":        modelSlug,
			"modality":          modality,
			"profile":           profileName,
			"profile_slug":      profileSlug,
			"price_seq_id":      priceSeqID,
			"pricing_variant":   pricingVariant,
			"price_type":        priceType,
			"price_unit":        priceUnit,
			"price":             price,
			"currency":          currency,
			"provider_price":    providerPrice,
			"provider_currency": providerCurrency,
			"price_metadata":    priceMetadata,
			"status":            status,
		})
	}
	writeJSON(w, http.StatusOK, response{"pricing": items})
}

func (a *App) companyUsage(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_user_id"})
		return
	}

	var companyID, companyName, userType string
	err := a.DB.QueryRowContext(r.Context(), `
		select coalesce(u.company_id, ''), coalesce(c.name, ''), u.user_type
		from sys_users u
		left join user_companies c on c.id = u.company_id
		where u.id = $1`, userID).Scan(&companyID, &companyName, &userType)
	if err != nil {
		writeJSON(w, http.StatusNotFound, response{"error": "user_not_found"})
		return
	}
	if companyID == "" || userType != "corporate_admin" {
		writeJSON(w, http.StatusForbidden, response{"error": "corporate_admin_required"})
		return
	}

	members, totalCredits, err := a.companyUsageMembers(r.Context(), companyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	models, err := a.companyUsageModels(r.Context(), companyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{"company": response{"id": companyID, "name": companyName}, "total_credits": totalCredits, "members": members, "models": models})
}

func (a *App) companyUsageMembers(ctx context.Context, companyID string) ([]response, int64, error) {
	rows, err := a.DB.QueryContext(ctx, `
		select u.id, u.email, u.name, u.user_type, coalesce(sum(ue.customer_charge), 0)
		from sys_users u
		left join user_usage_events ue on ue.actor_user_id = u.id
		where u.company_id = $1
		group by u.id, u.email, u.name, u.user_type
		order by coalesce(sum(ue.customer_charge), 0) desc, u.name`, companyID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []response{}
	var total int64
	for rows.Next() {
		var id, email, name, userType string
		var credits int64
		if err := rows.Scan(&id, &email, &name, &userType, &credits); err != nil {
			return nil, 0, err
		}
		total += credits
		items = append(items, response{"id": id, "email": email, "name": name, "user_type": userType, "credits_used": credits})
	}
	return items, total, rows.Err()
}

func (a *App) companyUsageModels(ctx context.Context, companyID string) ([]response, error) {
	rows, err := a.DB.QueryContext(ctx, `
		select ue.model_slug, coalesce(m.name, ue.model_slug), coalesce(m.modality, ''), sum(ue.customer_charge)
		from user_usage_events ue
		join sys_users u on u.id = ue.actor_user_id
		left join sys_models m on m.slug = ue.model_slug
		where u.company_id = $1
		group by ue.model_slug, m.name, m.modality
		order by sum(ue.customer_charge) desc`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var slug, name, modality string
		var credits int64
		if err := rows.Scan(&slug, &name, &modality, &credits); err != nil {
			return nil, err
		}
		items = append(items, response{"model_slug": slug, "model": name, "modality": modality, "credits_used": credits})
	}
	return items, rows.Err()
}

func (a *App) userCreditUsage(w http.ResponseWriter, r *http.Request) {
	rangeDays := parseRangeDays(r.URL.Query().Get("range"))
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if userID == "" {
		userID = "user-yong-zhao"
	}
	if _, err := a.resolveUsageUserID(r.Context(), userID); err != nil {
		userID = "user-yong-zhao"
	}

	usage, totalTokens, topModel, err := a.userTokenUsageRows(r.Context(), userID, rangeDays)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	if len(usage) == 0 && userID != "user-yong-zhao" {
		userID = "user-yong-zhao"
		usage, totalTokens, topModel, err = a.userTokenUsageRows(r.Context(), userID, rangeDays)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
	}
	purchases, creditsBought, err := a.userCreditPurchases(r.Context(), userID, rangeDays)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{
		"user_id":        userID,
		"range_days":     rangeDays,
		"total_tokens":   totalTokens,
		"credits_bought": creditsBought,
		"top_model":      topModel,
		"usage":          usage,
		"purchases":      purchases,
	})
}

func (a *App) resolveUsageUserID(ctx context.Context, value string) (string, error) {
	var id string
	err := a.DB.QueryRowContext(ctx, `
		select id
		from sys_users
		where id = $1 or lower(email) = lower($1) or lower(name) = lower($1)
		limit 1`, value).Scan(&id)
	return id, err
}

func (a *App) userTokenUsageRows(ctx context.Context, userID string, rangeDays int) ([]response, int64, string, error) {
	rows, err := a.DB.QueryContext(ctx, `
		select to_char(date(ue.created_at), 'YYYY-MM-DD'), ue.model_slug, coalesce(m.name, ue.model_slug),
			coalesce(m.modality, ''), sum(ue.input_tokens), sum(ue.output_tokens), sum(ue.customer_charge)
		from user_usage_events ue
		left join sys_models m on m.slug = ue.model_slug
		where ue.actor_user_id = $1
			and ue.created_at >= current_timestamp - ($2::int * interval '1 day')
		group by date(ue.created_at), ue.model_slug, m.name, m.modality
		order by date(ue.created_at) asc, sum(ue.input_tokens + ue.output_tokens) desc`, userID, rangeDays)
	if err != nil {
		return nil, 0, "", err
	}
	defer rows.Close()
	items := []response{}
	totalsByModel := map[string]int64{}
	var totalTokens int64
	for rows.Next() {
		var date, slug, name, modality string
		var inputTokens, outputTokens, customerCharge int64
		if err := rows.Scan(&date, &slug, &name, &modality, &inputTokens, &outputTokens, &customerCharge); err != nil {
			return nil, 0, "", err
		}
		rowTotal := inputTokens + outputTokens
		totalTokens += rowTotal
		totalsByModel[name] += rowTotal
		items = append(items, response{"date": date, "model": name, "model_slug": slug, "modality": modality, "input_tokens": inputTokens, "output_tokens": outputTokens, "customer_charge": customerCharge})
	}
	topModel := ""
	var topTokens int64
	for model, tokens := range totalsByModel {
		if tokens > topTokens {
			topModel = model
			topTokens = tokens
		}
	}
	if topModel == "" {
		topModel = "-"
	}
	return items, totalTokens, topModel, rows.Err()
}

func (a *App) userCreditPurchases(ctx context.Context, userID string, rangeDays int) ([]response, int64, error) {
	rows, err := a.DB.QueryContext(ctx, `
		select to_char(date(created_at), 'YYYY-MM-DD'), credits, amount_cents, currency, status
		from user_credit_purchases
		where user_id = $1
			and created_at >= current_timestamp - ($2::int * interval '1 day')
		order by created_at desc`, userID, rangeDays)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []response{}
	var totalCredits int64
	for rows.Next() {
		var date, currency, status string
		var credits, amountCents int64
		if err := rows.Scan(&date, &credits, &amountCents, &currency, &status); err != nil {
			return nil, 0, err
		}
		totalCredits += credits
		items = append(items, response{"date": date, "credits": credits, "amount_cents": amountCents, "currency": currency, "status": status})
	}
	return items, totalCredits, rows.Err()
}

func parseRangeDays(value string) int {
	switch strings.TrimSpace(value) {
	case "7":
		return 7
	case "90":
		return 90
	default:
		return 30
	}
}

func (a *App) purchaseCredits(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID        string `json:"user_id"`
		AmountCents   int64  `json:"amount_cents"`
		Credits       int64  `json:"credits"`
		PaymentMethod string `json:"payment_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	req.UserID = strings.TrimSpace(req.UserID)
	req.PaymentMethod = strings.TrimSpace(req.PaymentMethod)
	if req.UserID == "" {
		req.UserID = "user-yong-zhao"
	}
	if req.AmountCents <= 0 && req.Credits <= 0 {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_payment_amount"})
		return
	}
	userID, err := a.resolveUsageUserID(r.Context(), req.UserID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, response{"error": "user_not_found"})
		return
	}
	if req.PaymentMethod == "" {
		req.PaymentMethod = "credit_card"
	}
	ctx := r.Context()
	ratio, err := a.creditRatio(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	amountCents := req.AmountCents
	if amountCents <= 0 {
		amountCents = int64(float64(req.Credits) / ratio * 100)
	}
	credits := int64(float64(amountCents) / 100 * ratio)
	if credits <= 0 {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_credit_amount"})
		return
	}
	paymentModeFallback := strings.TrimSpace(os.Getenv("PAYMENT_PROVIDER_MODE"))
	if paymentModeFallback == "" {
		paymentModeFallback = "mock"
	}
	paymentMode, err := a.configValue(r.Context(), "payment_provider_mode", paymentModeFallback)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	paymentMode = strings.ToLower(paymentMode)
	if paymentMode == "stripe" {
		result, err := a.createStripeCreditPurchase(ctx, userID, req.PaymentMethod, amountCents, credits, ratio)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, response{"error": "stripe_checkout_failed", "message": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, result)
		return
	}
	mockEnabled, err := a.configBool(r.Context(), "payment_mock_enabled", true)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	if paymentMode != "mock" {
		writeJSON(w, http.StatusNotImplemented, response{"error": "payment_provider_not_configured", "payment_provider_mode": paymentMode})
		return
	}
	if !mockEnabled {
		writeJSON(w, http.StatusServiceUnavailable, response{"error": "mock_payments_disabled"})
		return
	}
	walletID, err := a.walletForUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	purchaseID := "purchase_" + randomHex(8)
	paymentID := "payment_" + randomHex(8)
	ledgerID := "ledger_" + randomHex(8)
	providerPaymentID := "fake_" + randomHex(8)
	metadata := fmt.Sprintf(`{"fake":true,"payment_provider_mode":%q,"payment_method":%q,"usd_to_credit_ratio":%g}`, paymentMode, req.PaymentMethod, ratio)
	if _, err := tx.ExecContext(r.Context(), `
		insert into user_credit_purchases(id, user_id, credits, amount_cents, currency, status, metadata)
		values($1, $2, $3, $4, 'USD', 'posted', $5)`, purchaseID, userID, credits, amountCents, metadata); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	if _, err := tx.ExecContext(r.Context(), `
		insert into user_payments(id, wallet_id, provider, provider_payment_id, amount_cents, currency, status, metadata)
		values($1, $2, $3, $4, $5, 'USD', 'succeeded', $6)`, paymentID, walletID, req.PaymentMethod, providerPaymentID, amountCents, metadata); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	if _, err := tx.ExecContext(r.Context(), `
		insert into user_ledger_transactions(id, wallet_id, transaction_type, amount, credit_type, status, reason, idempotency_key, metadata)
		values($1, $2, 'purchase', $3, 'paid', 'posted', 'mock credit purchase', $4, $5)`, ledgerID, walletID, credits, purchaseID, metadata); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	if _, err := tx.ExecContext(r.Context(), `
		update user_wallets
		set paid_credits = paid_credits + $1, updated_at = current_timestamp
		where id = $2`, credits, walletID); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, response{"purchase_id": purchaseID, "wallet_id": walletID, "credits": credits, "amount_cents": amountCents, "currency": "USD", "payment_provider_mode": paymentMode, "provider_payment_id": providerPaymentID, "status": "posted"})
}

type stripeCheckoutSession struct {
	ID            string `json:"id"`
	URL           string `json:"url"`
	PaymentStatus string `json:"payment_status"`
}

func (a *App) createStripeCreditPurchase(ctx context.Context, userID, paymentMethod string, amountCents, credits int64, ratio float64) (response, error) {
	secretKey := strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
	if secretKey == "" {
		return nil, errors.New("STRIPE_SECRET_KEY is not set")
	}
	walletID, err := a.walletForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	purchaseID := "purchase_" + randomHex(8)
	paymentID := "payment_" + randomHex(8)
	metadata := fmt.Sprintf(`{"fake":false,"payment_provider_mode":"stripe","payment_method":%q,"usd_to_credit_ratio":%g}`, paymentMethod, ratio)
	session, err := a.createStripeCheckoutSession(ctx, secretKey, purchaseID, walletID, userID, amountCents, credits)
	if err != nil {
		return nil, err
	}

	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		insert into user_credit_purchases(id, user_id, credits, amount_cents, currency, status, metadata)
		values($1, $2, $3, $4, 'USD', 'pending', $5)`, purchaseID, userID, credits, amountCents, metadata); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		insert into user_payments(id, wallet_id, provider, provider_payment_id, amount_cents, currency, status, metadata)
		values($1, $2, 'stripe', $3, $4, 'USD', 'pending', $5)`, paymentID, walletID, session.ID, amountCents, metadata); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return response{
		"purchase_id":           purchaseID,
		"wallet_id":             walletID,
		"credits":               credits,
		"amount_cents":          amountCents,
		"currency":              "USD",
		"payment_provider_mode": "stripe",
		"provider_payment_id":   session.ID,
		"checkout_url":          session.URL,
		"status":                "pending",
	}, nil
}

func (a *App) createStripeCheckoutSession(ctx context.Context, secretKey, purchaseID, walletID, userID string, amountCents, credits int64) (stripeCheckoutSession, error) {
	publicURL := strings.TrimRight(a.Config.PublicURL, "/")
	if publicURL == "" {
		publicURL = "http://localhost:3000"
	}
	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("success_url", publicURL+"/#credit-usage")
	form.Set("cancel_url", publicURL+"/#pricing")
	form.Set("client_reference_id", purchaseID)
	form.Set("line_items[0][quantity]", "1")
	form.Set("line_items[0][price_data][currency]", "usd")
	form.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(amountCents, 10))
	form.Set("line_items[0][price_data][product_data][name]", "Model Market credits")
	form.Set("line_items[0][price_data][product_data][description]", fmt.Sprintf("%d credits", credits))
	form.Set("metadata[purchase_id]", purchaseID)
	form.Set("metadata[wallet_id]", walletID)
	form.Set("metadata[user_id]", userID)
	form.Set("metadata[credits]", strconv.FormatInt(credits, 10))
	form.Set("metadata[amount_cents]", strconv.FormatInt(amountCents, 10))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.stripe.com/v1/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return stripeCheckoutSession{}, err
	}
	req.Header.Set("Authorization", "Bearer "+secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := a.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return stripeCheckoutSession{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return stripeCheckoutSession{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return stripeCheckoutSession{}, fmt.Errorf("stripe status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var session stripeCheckoutSession
	if err := json.Unmarshal(body, &session); err != nil {
		return stripeCheckoutSession{}, err
	}
	if session.ID == "" || session.URL == "" {
		return stripeCheckoutSession{}, errors.New("stripe checkout session missing id or url")
	}
	return session, nil
}

func (a *App) stripeWebhook(w http.ResponseWriter, r *http.Request) {
	secret := strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if secret == "" {
		writeJSON(w, http.StatusServiceUnavailable, response{"error": "STRIPE_WEBHOOK_SECRET is not set"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_body"})
		return
	}
	if !validStripeSignature(body, r.Header.Get("Stripe-Signature"), secret, 5*time.Minute) {
		writeJSON(w, http.StatusUnauthorized, response{"error": "invalid_signature"})
		return
	}
	var event struct {
		ID   string          `json:"id"`
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	if event.Type != "checkout.session.completed" {
		writeJSON(w, http.StatusOK, response{"status": "ignored", "event_type": event.Type})
		return
	}
	var data struct {
		Object struct {
			ID            string            `json:"id"`
			PaymentStatus string            `json:"payment_status"`
			PaymentIntent string            `json:"payment_intent"`
			Metadata      map[string]string `json:"metadata"`
		} `json:"object"`
	}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_session"})
		return
	}
	if data.Object.PaymentStatus != "" && data.Object.PaymentStatus != "paid" {
		writeJSON(w, http.StatusOK, response{"status": "ignored", "payment_status": data.Object.PaymentStatus})
		return
	}
	result, err := a.postStripeCreditPurchase(r.Context(), data.Object.ID, data.Object.PaymentIntent, data.Object.Metadata)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func validStripeSignature(payload []byte, header, secret string, tolerance time.Duration) bool {
	parts := strings.Split(header, ",")
	var timestamp string
	signatures := []string{}
	for _, part := range parts {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			timestamp = value
		case "v1":
			signatures = append(signatures, value)
		}
	}
	if timestamp == "" || len(signatures) == 0 {
		return false
	}
	unix, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if tolerance > 0 && time.Since(time.Unix(unix, 0)) > tolerance {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	for _, signature := range signatures {
		if hmac.Equal([]byte(expected), []byte(signature)) {
			return true
		}
	}
	return false
}

func (a *App) postStripeCreditPurchase(ctx context.Context, sessionID, paymentIntent string, metadata map[string]string) (response, error) {
	purchaseID := metadata["purchase_id"]
	walletID := metadata["wallet_id"]
	credits, _ := strconv.ParseInt(metadata["credits"], 10, 64)
	if purchaseID == "" || walletID == "" || credits <= 0 || sessionID == "" {
		return nil, errors.New("stripe webhook missing purchase metadata")
	}
	idempotencyKey := "stripe_checkout_" + sessionID
	var existing string
	if err := a.DB.QueryRowContext(ctx, `select id from user_ledger_transactions where idempotency_key = $1`, idempotencyKey).Scan(&existing); err == nil {
		return response{"status": "already_posted", "ledger_id": existing, "purchase_id": purchaseID}, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ledgerID := "ledger_" + randomHex(8)
	providerID := sessionID
	if paymentIntent != "" {
		providerID = paymentIntent
	}
	eventMetadata := fmt.Sprintf(`{"payment_provider_mode":"stripe","checkout_session_id":%q,"payment_intent":%q}`, sessionID, paymentIntent)
	if _, err := tx.ExecContext(ctx, `
		update user_credit_purchases
		set status = 'posted', metadata = $2
		where id = $1`, purchaseID, eventMetadata); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		update user_payments
		set status = 'succeeded', provider_payment_id = $2, metadata = $3
		where provider = 'stripe' and provider_payment_id = $1`, sessionID, providerID, eventMetadata); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		insert into user_ledger_transactions(id, wallet_id, transaction_type, amount, credit_type, status, reason, idempotency_key, metadata)
		values($1, $2, 'purchase', $3, 'paid', 'posted', 'stripe credit purchase', $4, $5)`, ledgerID, walletID, credits, idempotencyKey, eventMetadata); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		update user_wallets
		set paid_credits = paid_credits + $1, updated_at = current_timestamp
		where id = $2`, credits, walletID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return response{"status": "posted", "ledger_id": ledgerID, "purchase_id": purchaseID, "credits": credits}, nil
}

func (a *App) creditRatio(ctx context.Context) (float64, error) {
	raw, err := a.configValue(ctx, "usd_to_credit_ratio", "1")
	if err != nil {
		return 1, err
	}
	var ratio float64
	if _, err := fmt.Sscanf(raw, "%f", &ratio); err != nil || ratio <= 0 {
		return 1, nil
	}
	return ratio, nil
}

func (a *App) configValue(ctx context.Context, key, fallback string) (string, error) {
	var raw string
	if err := a.DB.QueryRowContext(ctx, `select conf_value from sys_config where conf_key = $1`, key).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fallback, nil
		}
		return fallback, err
	}
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	return strings.TrimSpace(raw), nil
}

func (a *App) configBool(ctx context.Context, key string, fallback bool) (bool, error) {
	raw, err := a.configValue(ctx, key, fmt.Sprintf("%t", fallback))
	if err != nil {
		return fallback, err
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "y", "on":
		return true, nil
	case "false", "0", "no", "n", "off":
		return false, nil
	default:
		return fallback, nil
	}
}

func (a *App) walletForUser(ctx context.Context, userID string) (string, error) {
	var walletID string
	err := a.DB.QueryRowContext(ctx, `
		select w.id
		from sys_users u
		join user_wallets w on w.company_id = u.company_id and w.project_id is null
		where u.id = $1 and u.company_id is not null
		limit 1`, userID).Scan(&walletID)
	if err == nil {
		return walletID, nil
	}
	err = a.DB.QueryRowContext(ctx, `
		select w.id
		from sys_memberships m
		join user_projects p on p.organization_id = m.organization_id
		join user_wallets w on w.project_id = p.id
		where m.user_id = $1
		order by p.created_at asc
		limit 1`, userID).Scan(&walletID)
	return walletID, err
}

func (a *App) projects(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.QueryContext(r.Context(), `
		select p.id, p.name, o.name, coalesce(w.paid_credits, 0), coalesce(w.promotional_credits, 0), coalesce(u.credits_used, 0)
		from user_projects p
		join sys_organizations o on o.id = p.organization_id
		left join user_wallets w on w.project_id = p.id
			or (p.company_id is not null and w.company_id = p.company_id and w.project_id is null)
		left join (
			select project_id, sum(customer_charge) as credits_used
			from user_usage_events
			group by project_id
		) u on u.project_id = p.id
		order by p.created_at asc`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, name, org string
		var paid, promo, creditsUsed int64
		if err := rows.Scan(&id, &name, &org, &paid, &promo, &creditsUsed); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{"id": id, "name": name, "organization": org, "paid_credits": paid, "promotional_credits": promo, "credits_used": creditsUsed})
	}
	writeJSON(w, http.StatusOK, response{"projects": items})
}

func (a *App) createProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_project_name"})
		return
	}
	id := "project_" + randomHex(8)
	slug := slugify(name) + "-" + randomHex(3)
	_, err := a.DB.ExecContext(r.Context(), `
		insert into user_projects(id, organization_id, name, slug, environment, retention_policy)
		values($1, 'org-demo', $2, $3, 'dev', '{"conversation_days":30,"asset_days":30}')`, id, name, slug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	_, _ = a.DB.ExecContext(r.Context(), `
		insert into user_wallets(id, project_id, paid_credits, promotional_credits)
		values($1, $2, 0, 1000)`, "wallet_"+randomHex(8), id)
	writeJSON(w, http.StatusCreated, response{"project": response{"id": id, "name": name, "organization": "Demo Organization", "paid_credits": 0, "promotional_credits": 1000, "credits_used": 0}})
}

func (a *App) conversations(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_project_id"})
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `
		select c.id, c.title, c.status, c.created_at, c.updated_at, count(m.id)
		from user_conversations c
		left join user_messages m on m.conversation_id = c.id
		where c.project_id = $1
		group by c.id, c.title, c.status, c.created_at, c.updated_at
		order by c.updated_at desc`, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, title, status string
		var createdAt, updatedAt time.Time
		var messageCount int64
		if err := rows.Scan(&id, &title, &status, &createdAt, &updatedAt, &messageCount); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{"id": id, "title": title, "status": status, "created_at": createdAt, "updated_at": updatedAt, "message_count": messageCount})
	}
	writeJSON(w, http.StatusOK, response{"conversations": items})
}

func (a *App) createConversation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string `json:"project_id"`
		Title     string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProjectID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_request"})
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "New conversation"
	}
	id := "conversation_" + randomHex(8)
	_, err := a.DB.ExecContext(r.Context(), `
		insert into user_conversations(id, project_id, title, status)
		values($1, $2, $3, 'active')`, id, req.ProjectID, title)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	branchID := "branch_" + randomHex(8)
	_, _ = a.DB.ExecContext(r.Context(), `
		insert into user_conversation_branches(id, conversation_id, parent_branch_id, name)
		values($1, $2, null, 'Main')`, branchID, id)
	writeJSON(w, http.StatusCreated, response{"conversation": response{"id": id, "title": title, "status": "active", "message_count": 0}, "branch_id": branchID})
}

func (a *App) conversationBranches(w http.ResponseWriter, r *http.Request) {
	conversationID := r.URL.Query().Get("conversation_id")
	if conversationID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_conversation_id"})
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `
		select b.id, b.conversation_id, coalesce(b.parent_branch_id, ''), b.name, b.created_at, count(m.id)
		from user_conversation_branches b
		left join user_messages m on m.branch_id = b.id
		where b.conversation_id = $1
		group by b.id, b.conversation_id, b.parent_branch_id, b.name, b.created_at
		order by case when b.parent_branch_id is null then 0 else 1 end, b.created_at asc, b.name asc, b.id asc`, conversationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, branchConversationID, parentBranchID, name string
		var createdAt time.Time
		var messageCount int64
		if err := rows.Scan(&id, &branchConversationID, &parentBranchID, &name, &createdAt, &messageCount); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{"id": id, "conversation_id": branchConversationID, "parent_branch_id": parentBranchID, "name": name, "created_at": createdAt, "message_count": messageCount})
	}
	writeJSON(w, http.StatusOK, response{"branches": items})
}

func (a *App) createConversationBranch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConversationID  string `json:"conversation_id"`
		SourceMessageID string `json:"source_message_id"`
		Name            string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ConversationID == "" || req.SourceMessageID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_branch_request"})
		return
	}

	var sourceBranchID sql.NullString
	var sourceCreatedAt time.Time
	var sourceRole, sourceID string
	err := a.DB.QueryRowContext(r.Context(), `
		select branch_id, created_at, role, id
		from user_messages
		where id = $1 and conversation_id = $2`, req.SourceMessageID, req.ConversationID).Scan(&sourceBranchID, &sourceCreatedAt, &sourceRole, &sourceID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, response{"error": "source_message_not_found"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Branch from " + sourceCreatedAt.Format("Jan 2 15:04")
	}
	branchID := "branch_" + randomHex(8)
	_, err = a.DB.ExecContext(r.Context(), `
		insert into user_conversation_branches(id, conversation_id, parent_branch_id, name)
		values($1, $2, $3, $4)`, branchID, req.ConversationID, nullableString(sourceBranchID), name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}

	sourceRank := messageRoleRank(sourceRole)
	whereBranch := "branch_id is null"
	args := []any{req.ConversationID, sourceCreatedAt, sourceRank, req.SourceMessageID}
	if sourceBranchID.Valid {
		whereBranch = "branch_id = $5"
		args = append(args, sourceBranchID.String)
	}
	rows, err := a.DB.QueryContext(r.Context(), `
		select role, content, model_profile_id, inference_request_id, customer_charge, provider_cost, metadata, created_at
		from user_messages
		where conversation_id = $1
			and `+whereBranch+`
			and (
				created_at < $2
				or (created_at = $2 and case role when 'user' then 0 when 'assistant' then 1 else 2 end < $3)
				or (created_at = $2 and case role when 'user' then 0 when 'assistant' then 1 else 2 end = $3 and id <= $4)
			)
		order by created_at asc, case role when 'user' then 0 when 'assistant' then 1 else 2 end, id asc`, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()

	type copiedMessage struct {
		role, content, metadata string
		modelProfileID          sql.NullString
		inferenceRequestID      sql.NullString
		customerCharge          int64
		providerCost            int64
		createdAt               time.Time
	}
	copies := []copiedMessage{}
	for rows.Next() {
		var msg copiedMessage
		if err := rows.Scan(&msg.role, &msg.content, &msg.modelProfileID, &msg.inferenceRequestID, &msg.customerCharge, &msg.providerCost, &msg.metadata, &msg.createdAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		copies = append(copies, msg)
	}
	for _, msg := range copies {
		_, err = a.DB.ExecContext(r.Context(), `
			insert into user_messages(id, conversation_id, branch_id, role, content, model_profile_id, inference_request_id, customer_charge, provider_cost, metadata, created_at)
			values($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			"message_"+randomHex(12), req.ConversationID, branchID, msg.role, msg.content, nullableString(msg.modelProfileID), nullableString(msg.inferenceRequestID), msg.customerCharge, msg.providerCost, msg.metadata, msg.createdAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
	}
	_, _ = a.DB.ExecContext(r.Context(), `update user_conversations set updated_at = current_timestamp where id = $1`, req.ConversationID)
	writeJSON(w, http.StatusCreated, response{"branch": response{"id": branchID, "conversation_id": req.ConversationID, "parent_branch_id": sourceBranchID.String, "name": name, "message_count": len(copies)}})
}

func (a *App) messages(w http.ResponseWriter, r *http.Request) {
	conversationID := r.URL.Query().Get("conversation_id")
	if conversationID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_conversation_id"})
		return
	}
	branchID := r.URL.Query().Get("branch_id")
	whereBranch := ""
	args := []any{conversationID}
	if branchID != "" {
		whereBranch = " and branch_id = $2"
		args = append(args, branchID)
	}
	rows, err := a.DB.QueryContext(r.Context(), `
		select id, role, content, coalesce(model_profile_id, ''), coalesce(branch_id, ''),
			coalesce(inference_request_id, ''), customer_charge, provider_cost, metadata, created_at
		from user_messages
		where conversation_id = $1
		`+whereBranch+`
		order by created_at asc, case role when 'user' then 0 when 'assistant' then 1 else 2 end, id asc`, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, role, content, modelProfileID, messageBranchID, inferenceRequestID, metadata string
		var customerCharge, providerCost int64
		var createdAt time.Time
		if err := rows.Scan(&id, &role, &content, &modelProfileID, &messageBranchID, &inferenceRequestID, &customerCharge, &providerCost, &metadata, &createdAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{"id": id, "role": role, "content": content, "model_profile_id": modelProfileID, "branch_id": messageBranchID, "inference_request_id": inferenceRequestID, "customer_charge": customerCharge, "provider_cost": providerCost, "metadata": metadata, "created_at": createdAt})
	}
	writeJSON(w, http.StatusOK, response{"messages": items})
}

func (a *App) assets(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_project_id"})
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `
		select id, coalesce(conversation_id, ''), coalesce(branch_id, ''), asset_type, asset_origin, storage_path, storage_provider,
			coalesce(bucket_name, ''), coalesce(object_key, ''), coalesce(download_url, ''),
			coalesce(mime_type, ''), coalesce(size_bytes, 0), coalesce(inference_request_id, ''),
			customer_charge, provider_cost, metadata, created_at
		from user_workbench_assets
		where project_id = $1
		order by created_at desc`, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, conversationID, branchID, assetType, assetOrigin, storagePath, storageProvider, bucketName, objectKey, downloadURL, mimeType, inferenceRequestID, metadata string
		var sizeBytes, customerCharge, providerCost int64
		var createdAt time.Time
		if err := rows.Scan(&id, &conversationID, &branchID, &assetType, &assetOrigin, &storagePath, &storageProvider, &bucketName, &objectKey, &downloadURL, &mimeType, &sizeBytes, &inferenceRequestID, &customerCharge, &providerCost, &metadata, &createdAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{"id": id, "conversation_id": conversationID, "branch_id": branchID, "asset_type": assetType, "asset_origin": assetOrigin, "storage_path": storagePath, "storage_provider": storageProvider, "bucket_name": bucketName, "object_key": objectKey, "download_url": downloadURL, "mime_type": mimeType, "size_bytes": sizeBytes, "inference_request_id": inferenceRequestID, "customer_charge": customerCharge, "provider_cost": providerCost, "metadata": metadata, "created_at": createdAt})
	}
	writeJSON(w, http.StatusOK, response{"assets": items})
}

func (a *App) createUploadIntent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID      string `json:"project_id"`
		ConversationID string `json:"conversation_id"`
		BranchID       string `json:"branch_id"`
		Filename       string `json:"filename"`
		ContentType    string `json:"content_type"`
		SizeBytes      int64  `json:"size_bytes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProjectID == "" || req.Filename == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_upload_request"})
		return
	}
	contentType := strings.TrimSpace(req.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	assetType := assetTypeForContentType(contentType)
	if assetType == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "unsupported_upload_type"})
		return
	}
	assetID := "asset_" + randomHex(12)
	filename := sanitizeFilename(req.Filename)
	if filename == "" {
		filename = "upload.bin"
	}
	bucket := a.Config.AssetBucket
	objectKey := strings.Trim(strings.Join([]string{a.Config.AppEnv, "projects", req.ProjectID, "uploads", assetID, filename}, "/"), "/")
	storagePath := "s3://" + bucket + "/" + objectKey
	downloadURL := assetDownloadURL(a.Config.AssetPublicURL, bucket, objectKey)
	uploadURL := downloadURL

	_, err := a.DB.ExecContext(r.Context(), `
		insert into user_workbench_assets(id, project_id, conversation_id, branch_id, asset_type, asset_origin, storage_path, storage_provider, bucket_name, object_key, download_url, mime_type, size_bytes, metadata)
		values($1, $2, nullif($3, ''), nullif($4, ''), $5, 'uploaded', $6, 's3', $7, $8, $9, $10, $11, $12)`,
		assetID, req.ProjectID, req.ConversationID, req.BranchID, assetType, storagePath, bucket, objectKey, downloadURL, contentType, req.SizeBytes, fmt.Sprintf(`{"filename":%q,"upload_mode":"presigned"}`, req.Filename))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}

	messageID := ""
	if req.ConversationID != "" {
		messageID = "message_" + randomHex(12)
		_, err = a.DB.ExecContext(r.Context(), `
			insert into user_messages(id, conversation_id, branch_id, role, content, model_profile_id, inference_request_id, customer_charge, provider_cost, metadata)
			values($1, $2, nullif($3, ''), 'user', $4, null, null, 0, 0, $5)`,
			messageID, req.ConversationID, req.BranchID, "Uploaded "+filename, fmt.Sprintf(`{"asset_id":%q,"filename":%q}`, assetID, req.Filename))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		_, err = a.DB.ExecContext(r.Context(), `insert into user_message_attachments(id, message_id, asset_id) values($1, $2, $3)`, "attachment_"+randomHex(12), messageID, assetID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		_, _ = a.DB.ExecContext(r.Context(), `update user_conversations set updated_at = current_timestamp where id = $1`, req.ConversationID)
	}

	writeJSON(w, http.StatusCreated, response{
		"asset": response{
			"id": assetID, "project_id": req.ProjectID, "conversation_id": req.ConversationID, "branch_id": req.BranchID, "asset_type": assetType, "asset_origin": "uploaded",
			"storage_path": storagePath, "storage_provider": "s3", "bucket_name": bucket, "object_key": objectKey,
			"download_url": downloadURL, "mime_type": contentType, "size_bytes": req.SizeBytes,
		},
		"message_id": messageID,
		"upload":     response{"method": "PUT", "url": uploadURL},
	})
}

func (a *App) apiKeys(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	rows, err := a.DB.QueryContext(r.Context(), `
		select id, name, prefix, status, created_at, revoked_at
		from user_api_keys
		where project_id = $1
		order by created_at desc`, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, name, prefix, status string
		var createdAt time.Time
		var revokedAt sql.NullTime
		if err := rows.Scan(&id, &name, &prefix, &status, &createdAt, &revokedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		item := response{"id": id, "name": name, "prefix": prefix, "status": status, "created_at": createdAt}
		if revokedAt.Valid {
			item["revoked_at"] = revokedAt.Time
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, response{"api_keys": items})
}

func (a *App) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProjectID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_request"})
		return
	}
	if req.Name == "" {
		req.Name = "Development key"
	}
	raw := "mk_" + randomHex(24)
	prefix := raw[:10]
	hash := hashAPIKey(raw)
	id := "key_" + randomHex(12)
	_, err := a.DB.ExecContext(r.Context(), `
		insert into user_api_keys(id, project_id, name, prefix, key_hash, scopes, status)
		values($1, $2, $3, $4, $5, $6, 'active')`, id, req.ProjectID, req.Name, prefix, hash, "models:read,chat:create")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, response{"id": id, "api_key": raw, "prefix": prefix})
}

func (a *App) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/api-keys/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_id"})
		return
	}
	_, err := a.DB.ExecContext(r.Context(), `update user_api_keys set status = 'revoked', revoked_at = current_timestamp where id = $1`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{"status": "revoked", "id": id})
}

func (a *App) chatCompletions(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.authenticateAPIKey(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, response{"error": err.Error()})
		return
	}
	var req struct {
		Model          string        `json:"model"`
		ConversationID string        `json:"conversation_id"`
		BranchID       string        `json:"branch_id"`
		Messages       []chatMessage `json:"messages"`
		Parameters     response      `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Model == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_request"})
		return
	}
	route, err := a.selectModelRoute(r.Context(), req.Model, "default")
	if err != nil {
		writeJSON(w, http.StatusNotFound, response{"error": "no_route_for_model", "model": req.Model})
		return
	}
	upstream, err := a.runChatUpstream(r.Context(), route, req.Messages)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, response{"error": "upstream_failed", "provider": route.ProviderSlug, "channel_id": route.ChannelID, "message": err.Error()})
		return
	}
	content := upstream.Content
	requestID := "req_" + randomHex(12)
	customerCharge := int64(1)
	providerCost := int64(0)
	metadata := response{"requested_model": req.Model, "upstream_model_id": route.UpstreamModelID, "route_group": route.RouteGroup}
	if len(req.Parameters) > 0 {
		metadata["parameters"] = req.Parameters
	}
	metadataRaw, _ := json.Marshal(metadata)
	_, err = a.DB.ExecContext(r.Context(), `
		insert into user_inference_requests(id, project_id, model_slug, model_profile_id, route_id, channel_id, provider_slug, status, input_units, output_units, customer_charge, provider_cost, margin, metadata)
		values($1, $2, $3, $4, $5, $6, $7, 'succeeded', $8, $9, $10, $11, $12, $13)`,
		requestID, projectID, route.ModelSlug, nullIfEmpty(route.ModelProfileID), route.ID, route.ChannelID, route.ProviderSlug, upstream.PromptTokens, upstream.CompletionTokens, customerCharge, providerCost, customerCharge-providerCost, string(metadataRaw))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	_, err = a.DB.ExecContext(r.Context(), `
		insert into user_provider_attempts(id, inference_request_id, provider_id, channel_id, route_id, status, latency_ms, provider_request_id, error_class, metadata)
		values($1, $2, $3, $4, $5, 'succeeded', $6, $7, null, $8)`,
		"attempt_"+randomHex(12), requestID, route.ProviderID, route.ChannelID, route.ID, upstream.LatencyMS, upstream.ProviderRequestID, fmt.Sprintf(`{"upstream_model_id":%q,"channel_name":%q,"channel_type":%q}`, route.UpstreamModelID, route.ChannelName, route.ChannelType))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	if req.ConversationID != "" && len(req.Messages) > 0 {
		last := req.Messages[len(req.Messages)-1]
		if err := a.saveConversationTurn(r.Context(), projectID, req.ConversationID, req.BranchID, requestID, req.Model, last.Content, content, customerCharge, providerCost); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, response{
		"id":      requestID,
		"object":  "chat.completion",
		"model":   route.UpstreamModelID,
		"route":   response{"id": route.ID, "channel_id": route.ChannelID, "channel_name": route.ChannelName, "provider": route.ProviderSlug, "requested_model": req.Model},
		"choices": []response{{"index": 0, "message": response{"role": "assistant", "content": content}, "finish_reason": "stop"}},
		"usage":   response{"prompt_tokens": upstream.PromptTokens, "completion_tokens": upstream.CompletionTokens, "total_tokens": upstream.PromptTokens + upstream.CompletionTokens},
	})
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type upstreamChatResult struct {
	Content           string
	PromptTokens      int
	CompletionTokens  int
	LatencyMS         int64
	ProviderRequestID string
}

func (a *App) runChatUpstream(ctx context.Context, route selectedModelRoute, messages []chatMessage) (upstreamChatResult, error) {
	if route.ChannelType == "google_gemini" {
		return a.callGeminiGenerateContent(ctx, route, messages)
	}
	content := "Mock response from " + route.UpstreamModelID + " via " + route.ChannelName
	if len(messages) > 0 {
		content = "Mock response to: " + messages[len(messages)-1].Content
	}
	return upstreamChatResult{
		Content:           content,
		PromptTokens:      len(messages),
		CompletionTokens:  len(content),
		LatencyMS:         route.ResponseTimeMS,
		ProviderRequestID: "mock-" + randomHex(8),
	}, nil
}

func (a *App) callGeminiGenerateContent(ctx context.Context, route selectedModelRoute, messages []chatMessage) (upstreamChatResult, error) {
	apiKeyName := strings.TrimSpace(route.CredentialRef)
	if apiKeyName == "" {
		apiKeyName = "GEMINI_API_KEY"
	}
	apiKey := strings.TrimSpace(os.Getenv(apiKeyName))
	if apiKey == "" {
		return upstreamChatResult{}, fmt.Errorf("%s is not set", apiKeyName)
	}

	type geminiPart struct {
		Text string `json:"text"`
	}
	type geminiContent struct {
		Role  string       `json:"role,omitempty"`
		Parts []geminiPart `json:"parts"`
	}
	payload := struct {
		SystemInstruction *struct {
			Parts []geminiPart `json:"parts"`
		} `json:"system_instruction,omitempty"`
		Contents []geminiContent `json:"contents"`
	}{}

	systemParts := []geminiPart{}
	for _, message := range messages {
		text := strings.TrimSpace(message.Content)
		if text == "" {
			continue
		}
		switch strings.ToLower(message.Role) {
		case "system":
			systemParts = append(systemParts, geminiPart{Text: text})
		case "assistant", "model":
			payload.Contents = append(payload.Contents, geminiContent{Role: "model", Parts: []geminiPart{{Text: text}}})
		default:
			payload.Contents = append(payload.Contents, geminiContent{Role: "user", Parts: []geminiPart{{Text: text}}})
		}
	}
	if len(systemParts) > 0 {
		payload.SystemInstruction = &struct {
			Parts []geminiPart `json:"parts"`
		}{Parts: systemParts}
	}
	if len(payload.Contents) == 0 {
		payload.Contents = []geminiContent{{Role: "user", Parts: []geminiPart{{Text: "Hello"}}}}
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return upstreamChatResult{}, err
	}
	baseURL := strings.TrimRight(route.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	endpoint := baseURL + "/models/" + route.UpstreamModelID + ":generateContent"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return upstreamChatResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", apiKey)

	client := a.Client
	if client == nil {
		client = http.DefaultClient
	}
	start := time.Now()
	resp, err := client.Do(httpReq)
	latencyMS := time.Since(start).Milliseconds()
	if err != nil {
		return upstreamChatResult{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return upstreamChatResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return upstreamChatResult{}, fmt.Errorf("gemini status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return upstreamChatResult{}, err
	}
	parts := []string{}
	for _, candidate := range parsed.Candidates {
		for _, part := range candidate.Content.Parts {
			if strings.TrimSpace(part.Text) != "" {
				parts = append(parts, part.Text)
			}
		}
		if len(parts) > 0 {
			break
		}
	}
	content := strings.TrimSpace(strings.Join(parts, "\n"))
	if content == "" {
		return upstreamChatResult{}, errors.New("gemini returned no text")
	}
	promptTokens := parsed.UsageMetadata.PromptTokenCount
	completionTokens := parsed.UsageMetadata.CandidatesTokenCount
	if promptTokens == 0 {
		promptTokens = len(messages)
	}
	if completionTokens == 0 {
		completionTokens = len(content)
	}
	return upstreamChatResult{
		Content:           content,
		PromptTokens:      promptTokens,
		CompletionTokens:  completionTokens,
		LatencyMS:         latencyMS,
		ProviderRequestID: resp.Header.Get("x-request-id"),
	}, nil
}

type selectedModelRoute struct {
	ID              string
	RouteGroup      string
	ModelSlug       string
	ModelProfileID  string
	UpstreamModelID string
	ProviderID      string
	ProviderSlug    string
	ChannelID       string
	ChannelName     string
	ChannelType     string
	BaseURL         string
	CredentialRef   string
	Priority        int64
	Weight          int64
	ResponseTimeMS  int64
}

func (a *App) selectModelRoute(ctx context.Context, requestedModel, routeGroup string) (selectedModelRoute, error) {
	rows, err := a.DB.QueryContext(ctx, `
		select r.id, r.route_group, m.slug, coalesce(mp.id, ''), r.upstream_model_id,
			p.id, p.slug, c.id, c.name, c.channel_type, coalesce(c.base_url, ''), coalesce(c.credential_ref, ''),
			r.priority, r.weight, coalesce(c.response_time_ms, 0)
		from sys_channel_model_routes r
		join sys_provider_channels c on c.id = r.channel_id
		join sys_providers p on p.id = c.provider_id
		join sys_models m on m.id = r.model_id
		left join sys_model_profiles mp on mp.id = r.model_profile_id
		where r.enabled = true
			and r.status = 'active'
			and c.status = 'active'
			and p.status = 'active'
			and r.route_group = $1
			and (m.slug = $2 or mp.slug = $2)
		order by r.priority desc, r.weight desc, r.id asc`, routeGroup, requestedModel)
	if err != nil {
		return selectedModelRoute{}, err
	}
	defer rows.Close()

	candidates := []selectedModelRoute{}
	var topPriority *int64
	for rows.Next() {
		var route selectedModelRoute
		if err := rows.Scan(&route.ID, &route.RouteGroup, &route.ModelSlug, &route.ModelProfileID, &route.UpstreamModelID, &route.ProviderID, &route.ProviderSlug, &route.ChannelID, &route.ChannelName, &route.ChannelType, &route.BaseURL, &route.CredentialRef, &route.Priority, &route.Weight, &route.ResponseTimeMS); err != nil {
			return selectedModelRoute{}, err
		}
		if topPriority == nil {
			value := route.Priority
			topPriority = &value
		}
		if route.Priority != *topPriority {
			break
		}
		candidates = append(candidates, route)
	}
	if err := rows.Err(); err != nil {
		return selectedModelRoute{}, err
	}
	if len(candidates) == 0 {
		return selectedModelRoute{}, sql.ErrNoRows
	}
	totalWeight := int64(0)
	for _, route := range candidates {
		if route.Weight <= 0 {
			totalWeight += 1
		} else {
			totalWeight += route.Weight
		}
	}
	pick := time.Now().UnixNano() % totalWeight
	for _, route := range candidates {
		weight := route.Weight
		if weight <= 0 {
			weight = 1
		}
		pick -= weight
		if pick < 0 {
			return route, nil
		}
	}
	return candidates[0], nil
}

func (a *App) saveConversationTurn(ctx context.Context, projectID, conversationID, branchID, inferenceRequestID, model, prompt, answer string, customerCharge, providerCost int64) error {
	var existing string
	if err := a.DB.QueryRowContext(ctx, `select id from user_conversations where id = $1 and project_id = $2`, conversationID, projectID).Scan(&existing); err != nil {
		return err
	}
	if branchID == "" {
		if err := a.DB.QueryRowContext(ctx, `
			select id
			from user_conversation_branches
			where conversation_id = $1
			order by created_at asc, id asc
			limit 1`, conversationID).Scan(&branchID); err != nil {
			return err
		}
	} else {
		var existingBranch string
		if err := a.DB.QueryRowContext(ctx, `select id from user_conversation_branches where id = $1 and conversation_id = $2`, branchID, conversationID).Scan(&existingBranch); err != nil {
			return err
		}
	}
	userMessageID := "message_" + randomHex(12)
	assistantMessageID := "message_" + randomHex(12)
	_, err := a.DB.ExecContext(ctx, `
		insert into user_messages(id, conversation_id, branch_id, role, content, model_profile_id, inference_request_id, customer_charge, provider_cost, metadata)
		values($1, $2, $3, 'user', $4, null, null, 0, 0, '{}'),
			($5, $2, $3, 'assistant', $6, null, $7, $8, $9, $10)`,
		userMessageID, conversationID, branchID, prompt, assistantMessageID, answer, inferenceRequestID, customerCharge, providerCost, fmt.Sprintf(`{"model":%q}`, model))
	if err != nil {
		return err
	}
	_, err = a.DB.ExecContext(ctx, `update user_conversations set updated_at = current_timestamp where id = $1`, conversationID)
	return err
}

func (a *App) authenticateAPIKey(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", errors.New("missing_api_key")
	}
	hash := hashAPIKey(strings.TrimPrefix(header, "Bearer "))
	var projectID string
	err := a.DB.QueryRowContext(r.Context(), `select project_id from user_api_keys where key_hash = $1 and status = 'active'`, hash).Scan(&projectID)
	if err != nil {
		return "", errors.New("invalid_api_key")
	}
	return projectID, nil
}

func nullableString(value sql.NullString) any {
	if value.Valid {
		return value.String
	}
	return nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func messageRoleRank(role string) int {
	if role == "user" {
		return 0
	}
	if role == "assistant" {
		return 1
	}
	return 2
}

func assetTypeForContentType(contentType string) string {
	lower := strings.ToLower(contentType)
	if strings.HasPrefix(lower, "image/") {
		return "upload_image"
	}
	if strings.HasPrefix(lower, "audio/") {
		return "upload_audio"
	}
	if strings.HasPrefix(lower, "video/") {
		return "upload_video"
	}
	return ""
}

func sanitizeFilename(filename string) string {
	filename = strings.TrimSpace(strings.ReplaceAll(filename, "\\", "/"))
	if slash := strings.LastIndex(filename, "/"); slash >= 0 {
		filename = filename[slash+1:]
	}
	var b strings.Builder
	for _, ch := range filename {
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '.' || ch == '-' || ch == '_' {
			b.WriteRune(ch)
			continue
		}
		b.WriteByte('-')
	}
	return strings.Trim(b.String(), "-")
}

func assetDownloadURL(publicBaseURL, bucket, objectKey string) string {
	if publicBaseURL != "" {
		return publicBaseURL + "/" + objectKey
	}
	return "https://" + bucket + ".s3.amazonaws.com/" + objectKey
}

func randomHex(bytesLen int) string {
	buf := make([]byte, bytesLen)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func hashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func hashPassword(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, ch := range value {
		if ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "project"
	}
	return slug
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
