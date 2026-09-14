package httpapi

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const githubStateCookie = "mm_github_oauth_state"

type githubProfile struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func githubOAuthConfig() (string, string, string, error) {
	id, secret, redirect := strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID")), strings.TrimSpace(os.Getenv("GITHUB_CLIENT_SECRET")), strings.TrimSpace(os.Getenv("GITHUB_REDIRECT_URI"))
	if id == "" || secret == "" || redirect == "" {
		return "", "", "", errors.New("github_oauth_not_configured")
	}
	p, err := url.Parse(redirect)
	if err != nil || p.Scheme == "" || p.Host == "" {
		return "", "", "", errors.New("invalid_github_oauth_config")
	}
	return id, secret, redirect, nil
}

func (a *App) githubOAuthStart(w http.ResponseWriter, r *http.Request) {
	id, _, redirect, err := githubOAuthConfig()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, response{"error": err.Error()})
		return
	}
	state := randomHex(32)
	http.SetCookie(w, &http.Cookie{Name: githubStateCookie, Value: state, Path: "/api/v1/auth/oauth/github", HttpOnly: true, Secure: strings.HasPrefix(strings.ToLower(redirect), "https://"), SameSite: http.SameSiteLaxMode, MaxAge: 600})
	q := url.Values{"client_id": {id}, "redirect_uri": {redirect}, "scope": {"read:user user:email"}, "state": {state}}
	http.Redirect(w, r, "https://github.com/login/oauth/authorize?"+q.Encode(), http.StatusFound)
}

func (a *App) githubOAuthCallback(w http.ResponseWriter, r *http.Request) {
	id, secret, redirect, err := githubOAuthConfig()
	secure := strings.HasPrefix(strings.ToLower(redirect), "https://")
	http.SetCookie(w, &http.Cookie{Name: githubStateCookie, Value: "", Path: "/api/v1/auth/oauth/github", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	if err != nil {
		a.redirectOAuthError(w, r, "github_not_configured")
		return
	}
	cookie, cookieErr := r.Cookie(githubStateCookie)
	state, code := r.URL.Query().Get("state"), strings.TrimSpace(r.URL.Query().Get("code"))
	if cookieErr != nil || state == "" || code == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(state)) != 1 {
		a.redirectOAuthError(w, r, "invalid_oauth_state")
		return
	}
	token, err := a.exchangeGithubCode(r.Context(), id, secret, redirect, code)
	if err != nil {
		a.redirectOAuthError(w, r, "token_exchange_failed")
		return
	}
	profile, err := a.fetchGithubProfile(r.Context(), token)
	if err != nil {
		a.redirectOAuthError(w, r, "invalid_identity")
		return
	}
	userID, err := a.linkGithubIdentity(r.Context(), profile)
	if err != nil {
		a.redirectOAuthError(w, r, "account_link_failed")
		return
	}
	loginCode := "oauth_" + randomHex(32)
	if _, err = a.DB.ExecContext(r.Context(), `insert into sys_oauth_login_codes(id,user_id,code_hash,provider,expires_at) values($1,$2,$3,'github',$4)`, "oauth_code_"+randomHex(12), userID, sessionTokenHash(loginCode), time.Now().UTC().Add(2*time.Minute)); err != nil {
		a.redirectOAuthError(w, r, "login_code_failed")
		return
	}
	target, err := url.Parse(a.Config.PublicURL)
	if err != nil || target.Scheme == "" || target.Host == "" {
		a.redirectOAuthError(w, r, "invalid_public_url")
		return
	}
	q := target.Query()
	q.Set("oauth_code", loginCode)
	q.Set("oauth_provider", "github")
	target.RawQuery = q.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

func (a *App) exchangeGithubCode(ctx context.Context, id, secret, redirect, code string) (string, error) {
	form := url.Values{"client_id": {id}, "client_secret": {secret}, "code": {code}, "redirect_uri": {redirect}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.oauthHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("github token endpoint returned %d", resp.StatusCode)
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil || out.AccessToken == "" {
		return "", errors.New("github token response missing access_token")
	}
	return out.AccessToken, nil
}

func (a *App) fetchGithubProfile(ctx context.Context, token string) (githubProfile, error) {
	var profile githubProfile
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return profile, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "model-market")
	resp, err := a.oauthHTTPClient().Do(req)
	if err != nil {
		return profile, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&profile) != nil || profile.ID == 0 {
		return githubProfile{}, errors.New("github profile request failed")
	}
	if strings.TrimSpace(profile.Email) == "" {
		email, emailErr := a.fetchGithubEmail(ctx, token)
		if emailErr == nil {
			profile.Email = email
		}
	}
	if strings.TrimSpace(profile.Email) == "" {
		return githubProfile{}, errors.New("github account must provide an email address")
	}
	profile.Email, profile.Name = strings.ToLower(strings.TrimSpace(profile.Email)), strings.TrimSpace(profile.Name)
	if profile.Name == "" {
		profile.Name = profile.Login
	}
	return profile, nil
}

func (a *App) fetchGithubEmail(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "model-market")
	resp, err := a.oauthHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("github email endpoint returned %d", resp.StatusCode)
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&emails); err != nil {
		return "", err
	}
	for _, item := range emails {
		if item.Primary && item.Verified {
			return strings.ToLower(strings.TrimSpace(item.Email)), nil
		}
	}
	for _, item := range emails {
		if item.Verified {
			return strings.ToLower(strings.TrimSpace(item.Email)), nil
		}
	}
	return "", errors.New("github account has no verified email")
}

func (a *App) linkGithubIdentity(ctx context.Context, profile githubProfile) (string, error) {
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	providerID := fmt.Sprintf("%d", profile.ID)
	var userID string
	err = tx.QueryRowContext(ctx, `select user_id from sys_oauth_accounts where provider='github' and provider_account_id=$1 for update`, providerID).Scan(&userID)
	if err == nil {
		_, err = tx.ExecContext(ctx, `update sys_oauth_accounts set email=$1,display_name=$2,avatar_url=$3,last_login_at=current_timestamp where provider='github' and provider_account_id=$4`, profile.Email, profile.Name, profile.AvatarURL, providerID)
		if err != nil {
			return "", err
		}
		return userID, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	err = tx.QueryRowContext(ctx, `select id from sys_users where lower(email)=lower($1) for update`, profile.Email).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		userID = "user_" + randomHex(8)
		orgID, projectID := "org_"+randomHex(8), "project_"+randomHex(8)
		if _, err = tx.ExecContext(ctx, `insert into sys_users(id,email,name,avatar_url,status,password_hash,user_type,ui_theme,language) values($1,$2,$3,$4,'active',null,'individual_consumer','Light','EN')`, userID, profile.Email, profile.Name, nullIfEmpty(profile.AvatarURL)); err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `insert into sys_organizations(id,name,slug,status) values($1,$2,$3,'active')`, orgID, profile.Name+" Workspace", "personal-"+randomHex(8)); err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `insert into sys_memberships(id,user_id,organization_id,role) values($1,$2,$3,'owner')`, "membership_"+randomHex(8), userID, orgID); err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `insert into user_projects(id,organization_id,name,slug,environment,retention_policy) values($1,$2,'My Project',$3,'dev','{"conversation_days":365,"asset_days":365}')`, projectID, orgID, "my-project-"+randomHex(6)); err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `insert into user_wallets(id,project_id,paid_credits,promotional_credits) values($1,$2,0,1000)`, "wallet_"+randomHex(8), projectID); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	if _, err = tx.ExecContext(ctx, `insert into sys_oauth_accounts(id,user_id,provider,provider_account_id,email,display_name,avatar_url,last_login_at) values($1,$2,'github',$3,$4,$5,$6,current_timestamp)`, "oauth_"+randomHex(8), userID, providerID, profile.Email, profile.Name, nullIfEmpty(profile.AvatarURL)); err != nil {
		return "", err
	}
	return userID, tx.Commit()
}
