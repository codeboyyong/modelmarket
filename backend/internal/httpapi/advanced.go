package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"
)

type requestWindow struct {
	started time.Time
	count   int64
}

type loginAttemptWindow struct {
	started     time.Time
	failures    int
	lockedUntil time.Time
}

func (a *App) checkLoginAllowed(key string) (time.Duration, bool) {
	now := time.Now()
	a.loginLimitMu.Lock()
	defer a.loginLimitMu.Unlock()
	if a.loginLimits == nil {
		a.loginLimits = map[string]*loginAttemptWindow{}
	}
	window := a.loginLimits[key]
	if window == nil {
		return 0, true
	}
	if window.lockedUntil.After(now) {
		return time.Until(window.lockedUntil), false
	}
	if now.Sub(window.started) >= 15*time.Minute {
		delete(a.loginLimits, key)
	}
	return 0, true
}

func (a *App) recordLoginFailure(key string) {
	now := time.Now()
	a.loginLimitMu.Lock()
	defer a.loginLimitMu.Unlock()
	if a.loginLimits == nil {
		a.loginLimits = map[string]*loginAttemptWindow{}
	}
	window := a.loginLimits[key]
	if window == nil || now.Sub(window.started) >= 15*time.Minute {
		window = &loginAttemptWindow{started: now}
		a.loginLimits[key] = window
	}
	window.failures++
	if window.failures >= 5 {
		window.lockedUntil = now.Add(15 * time.Minute)
	}
}

func (a *App) clearLoginFailures(key string) {
	a.loginLimitMu.Lock()
	defer a.loginLimitMu.Unlock()
	delete(a.loginLimits, key)
}

var organizationRoles = map[string]bool{
	"owner": true, "admin": true, "billing_admin": true, "developer": true,
	"analyst": true, "read_only": true, "provider_admin": true,
}

// enforceRequestLimit provides a process-local limiter for the single-process
// deployment. The key and policy live at API-key level so this can be replaced
// by a Redis counter without changing handler behavior.
func (a *App) enforceRequestLimit(key string, limit int64) (int64, time.Duration, error) {
	if limit <= 0 {
		return 0, 0, nil
	}
	now := time.Now()
	a.rateLimitMu.Lock()
	defer a.rateLimitMu.Unlock()
	if a.rateLimits == nil {
		a.rateLimits = map[string]*requestWindow{}
	}
	window := a.rateLimits[key]
	if window == nil || now.Sub(window.started) >= time.Minute {
		window = &requestWindow{started: now}
		a.rateLimits[key] = window
	}
	if window.count >= limit {
		return 0, time.Until(window.started.Add(time.Minute)), errors.New("rate_limit_exceeded")
	}
	window.count++
	return limit - window.count, time.Until(window.started.Add(time.Minute)), nil
}

func requestIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func ipAllowed(ip, csv string) bool {
	if strings.TrimSpace(csv) == "" {
		return true
	}
	for _, allowed := range strings.Split(csv, ",") {
		if strings.TrimSpace(allowed) == ip {
			return true
		}
	}
	return false
}

func (a *App) organizationMembers(w http.ResponseWriter, r *http.Request) {
	organizationID := strings.TrimSpace(r.URL.Query().Get("organization_id"))
	if organizationID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_organization_id"})
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `
		select m.id, u.id, u.email, u.name, m.role, m.created_at
		from sys_memberships m join sys_users u on u.id = m.user_id
		where m.organization_id = $1 order by u.name`, organizationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var membershipID, userID, email, name, role string
		var createdAt time.Time
		if err := rows.Scan(&membershipID, &userID, &email, &name, &role, &createdAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{"id": membershipID, "user_id": userID, "email": email, "name": name, "role": role, "created_at": createdAt})
	}
	writeJSON(w, http.StatusOK, response{"members": items})
}

func (a *App) updateOrganizationMember(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrganizationID string `json:"organization_id"`
		UserID         string `json:"user_id"`
		Role           string `json:"role"`
		Remove         bool   `json:"remove"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.OrganizationID == "" || req.UserID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_request"})
		return
	}
	if req.Remove {
		result, err := a.DB.ExecContext(r.Context(), `delete from sys_memberships where organization_id = $1 and user_id = $2 and role <> 'owner'`, req.OrganizationID, req.UserID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		count, _ := result.RowsAffected()
		if count == 0 {
			writeJSON(w, http.StatusConflict, response{"error": "member_not_removed", "message": "owners cannot be removed"})
			return
		}
		writeJSON(w, http.StatusOK, response{"status": "removed"})
		return
	}
	if !organizationRoles[req.Role] {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_role"})
		return
	}
	result, err := a.DB.ExecContext(r.Context(), `update sys_memberships set role = $1 where organization_id = $2 and user_id = $3`, req.Role, req.OrganizationID, req.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		writeJSON(w, http.StatusNotFound, response{"error": "membership_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, response{"status": "updated", "role": req.Role})
}

func (a *App) organizationInvitations(w http.ResponseWriter, r *http.Request) {
	organizationID := strings.TrimSpace(r.URL.Query().Get("organization_id"))
	rows, err := a.DB.QueryContext(r.Context(), `select id, email, role, status, expires_at, created_at from user_organization_invitations where organization_id = $1 order by created_at desc`, organizationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, email, role, status string
		var expiresAt, createdAt time.Time
		if err := rows.Scan(&id, &email, &role, &status, &expiresAt, &createdAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{"id": id, "email": email, "role": role, "status": status, "expires_at": expiresAt, "created_at": createdAt})
	}
	writeJSON(w, http.StatusOK, response{"invitations": items})
}

func (a *App) createOrganizationInvitation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrganizationID string `json:"organization_id"`
		Email          string `json:"email"`
		Role           string `json:"role"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.OrganizationID == "" || strings.TrimSpace(req.Email) == "" || !organizationRoles[req.Role] {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_invitation"})
		return
	}
	rawToken := "invite_" + randomHex(24)
	id := "invite_" + randomHex(12)
	_, err := a.DB.ExecContext(r.Context(), `insert into user_organization_invitations(id, organization_id, email, role, token_hash, expires_at) values($1, $2, lower($3), $4, $5, current_timestamp + interval '7 days')`, id, req.OrganizationID, req.Email, req.Role, hashAPIKey(rawToken))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, response{"id": id, "invitation_token": rawToken, "expires_in": 604800})
}

func (a *App) acceptOrganizationInvitation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token  string `json:"token"`
		UserID string `json:"user_id"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.Token) == "" || strings.TrimSpace(req.UserID) == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_invitation_acceptance"})
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	var invitationID, organizationID, email, role string
	var expiresAt time.Time
	err = tx.QueryRowContext(r.Context(), `select id, organization_id, email, role, expires_at from user_organization_invitations where token_hash = $1 and status = 'pending' for update`, hashAPIKey(req.Token)).Scan(&invitationID, &organizationID, &email, &role, &expiresAt)
	if err != nil || !expiresAt.After(time.Now()) {
		writeJSON(w, http.StatusGone, response{"error": "invitation_invalid_or_expired"})
		return
	}
	var userEmail string
	if err := tx.QueryRowContext(r.Context(), `select email from sys_users where id = $1`, req.UserID).Scan(&userEmail); err != nil || !strings.EqualFold(userEmail, email) {
		writeJSON(w, http.StatusForbidden, response{"error": "invitation_email_mismatch"})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `insert into sys_memberships(id, user_id, organization_id, role) values($1,$2,$3,$4) on conflict (user_id, organization_id) do update set role = excluded.role`, "membership_"+randomHex(8), req.UserID, organizationID, role); err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `update user_organization_invitations set status = 'accepted', accepted_at = current_timestamp where id = $1`, invitationID); err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	if err = tx.Commit(); err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{"status": "accepted", "organization_id": organizationID, "role": role})
}

func (a *App) rotateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		KeyID string `json:"key_id"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.KeyID) == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_request"})
		return
	}
	raw := "mk_" + randomHex(24)
	prefix := raw[:10]
	result, err := a.DB.ExecContext(r.Context(), `update user_api_keys set key_hash = $1, prefix = $2, last_used_at = null where id = $3 and status = 'active'`, hashAPIKey(raw), prefix, req.KeyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		writeJSON(w, http.StatusNotFound, response{"error": "api_key_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, response{"id": req.KeyID, "api_key": raw, "prefix": prefix})
}

func (a *App) payments(w http.ResponseWriter, r *http.Request) {
	walletID := strings.TrimSpace(r.URL.Query().Get("wallet_id"))
	rows, err := a.DB.QueryContext(r.Context(), `select id, provider, coalesce(provider_payment_id, ''), amount_cents, refunded_amount_cents, currency, status, created_at, updated_at from user_payments where wallet_id = $1 order by created_at desc`, walletID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, provider, providerID, currency, status string
		var amount, refunded int64
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &provider, &providerID, &amount, &refunded, &currency, &status, &createdAt, &updatedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{"id": id, "provider": provider, "provider_payment_id": providerID, "amount_cents": amount, "refunded_amount_cents": refunded, "currency": currency, "status": status, "created_at": createdAt, "updated_at": updatedAt})
	}
	writeJSON(w, http.StatusOK, response{"payments": items})
}

func (a *App) refundPayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PaymentID   string `json:"payment_id"`
		AmountCents int64  `json:"amount_cents"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.PaymentID == "" || req.AmountCents <= 0 {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_refund"})
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	var walletID, status string
	var amount, refunded int64
	if err := tx.QueryRowContext(r.Context(), `select wallet_id, amount_cents, refunded_amount_cents, status from user_payments where id = $1 for update`, req.PaymentID).Scan(&walletID, &amount, &refunded, &status); err != nil {
		writeJSON(w, http.StatusNotFound, response{"error": "payment_not_found"})
		return
	}
	if req.AmountCents > amount-refunded {
		writeJSON(w, http.StatusConflict, response{"error": "refund_exceeds_payment"})
		return
	}
	ratio, err := a.creditRatio(r.Context())
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	credits := int64(float64(req.AmountCents) / 100 * ratio)
	result, err := tx.ExecContext(r.Context(), `update user_wallets set paid_credits = paid_credits - $1, updated_at = current_timestamp where id = $2 and paid_credits >= $1`, credits, walletID)
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		writeJSON(w, http.StatusConflict, response{"error": "insufficient_paid_credits_for_refund"})
		return
	}
	newRefunded := refunded + req.AmountCents
	newStatus := "partially_refunded"
	if newRefunded == amount {
		newStatus = "refunded"
	}
	if _, err = tx.ExecContext(r.Context(), `update user_payments set refunded_amount_cents = $1, status = $2, updated_at = current_timestamp where id = $3`, newRefunded, newStatus, req.PaymentID); err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `insert into user_ledger_transactions(id, wallet_id, transaction_type, amount, credit_type, status, reason, idempotency_key, metadata) values($1, $2, 'refund', $3, 'paid', 'posted', 'payment refund', $4, '{}')`, "ledger_"+randomHex(8), walletID, -credits, "refund_"+req.PaymentID+"_"+randomHex(6)); err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	if err = tx.Commit(); err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{"payment_id": req.PaymentID, "status": newStatus, "refunded_amount_cents": newRefunded, "credits_reversed": credits})
}

// settleProjectCharge atomically consumes promotional credits first and then
// paid credits. The row lock prevents two requests from overspending a wallet.
func (a *App) settleProjectCharge(ctx context.Context, projectID, requestID string, credits int64) error {
	if credits <= 0 {
		return nil
	}
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var walletID string
	var paid, promotional int64
	err = tx.QueryRowContext(ctx, `select id, paid_credits, promotional_credits from user_wallets where project_id = $1 for update`, projectID).Scan(&walletID, &paid, &promotional)
	if err != nil {
		return err
	}
	if paid+promotional < credits {
		return errors.New("insufficient_credits")
	}
	promoCharge := credits
	if promoCharge > promotional {
		promoCharge = promotional
	}
	paidCharge := credits - promoCharge
	if _, err = tx.ExecContext(ctx, `update user_wallets set promotional_credits = promotional_credits - $1, paid_credits = paid_credits - $2, updated_at = current_timestamp where id = $3`, promoCharge, paidCharge, walletID); err != nil {
		return err
	}
	if promoCharge > 0 {
		if _, err = tx.ExecContext(ctx, `insert into user_ledger_transactions(id, wallet_id, transaction_type, amount, credit_type, status, reason, idempotency_key, metadata) values($1,$2,'usage',$3,'promotional','posted','inference charge',$4,'{}')`, "ledger_"+randomHex(8), walletID, -promoCharge, "usage_promo_"+requestID); err != nil {
			return err
		}
	}
	if paidCharge > 0 {
		if _, err = tx.ExecContext(ctx, `insert into user_ledger_transactions(id, wallet_id, transaction_type, amount, credit_type, status, reason, idempotency_key, metadata) values($1,$2,'usage',$3,'paid','posted','inference charge',$4,'{}')`, "ledger_"+randomHex(8), walletID, -paidCharge, "usage_paid_"+requestID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func nullTime(value sql.NullTime) any {
	if value.Valid {
		return value.Time
	}
	return nil
}
