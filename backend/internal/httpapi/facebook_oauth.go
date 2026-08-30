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

const facebookStateCookie = "mm_facebook_oauth_state"

type facebookProfile struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"picture"`
}

func (a *App) facebookOAuthStart(w http.ResponseWriter, r *http.Request) {
	clientID, _, redirectURI, graphVersion, err := facebookOAuthConfig()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, response{"error": err.Error()})
		return
	}
	state := randomHex(32)
	setFacebookOAuthCookie(w, state, strings.HasPrefix(strings.ToLower(redirectURI), "https://"), 600)
	query := url.Values{"client_id": {clientID}, "redirect_uri": {redirectURI}, "response_type": {"code"}, "scope": {"email,public_profile"}, "state": {state}}
	http.Redirect(w, r, "https://www.facebook.com/"+graphVersion+"/dialog/oauth?"+query.Encode(), http.StatusFound)
}

func (a *App) facebookOAuthCallback(w http.ResponseWriter, r *http.Request) {
	clientID, clientSecret, redirectURI, graphVersion, err := facebookOAuthConfig()
	if err != nil {
		a.redirectOAuthError(w, r, "facebook_not_configured")
		return
	}
	secure := strings.HasPrefix(strings.ToLower(redirectURI), "https://")
	defer setFacebookOAuthCookie(w, "", secure, -1)
	if providerError := strings.TrimSpace(r.URL.Query().Get("error")); providerError != "" {
		a.redirectOAuthError(w, r, providerError)
		return
	}
	stateCookie, cookieErr := r.Cookie(facebookStateCookie)
	state, code := r.URL.Query().Get("state"), strings.TrimSpace(r.URL.Query().Get("code"))
	if cookieErr != nil || state == "" || code == "" || subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(state)) != 1 {
		a.redirectOAuthError(w, r, "invalid_oauth_state")
		return
	}
	accessToken, err := a.exchangeFacebookCode(r.Context(), clientID, clientSecret, redirectURI, graphVersion, code)
	if err != nil || a.validateFacebookToken(r.Context(), clientID, clientSecret, graphVersion, accessToken) != nil {
		a.redirectOAuthError(w, r, "token_exchange_failed")
		return
	}
	profile, err := a.fetchFacebookProfile(r.Context(), graphVersion, accessToken)
	if err != nil {
		a.redirectOAuthError(w, r, "invalid_identity")
		return
	}
	userID, err := a.linkFacebookIdentity(r.Context(), profile)
	if err != nil {
		a.redirectOAuthError(w, r, "account_link_failed")
		return
	}
	loginCode := "oauth_" + randomHex(32)
	_, err = a.DB.ExecContext(r.Context(), `insert into sys_oauth_login_codes(id,user_id,code_hash,provider,expires_at) values($1,$2,$3,'facebook',$4)`, "oauth_code_"+randomHex(12), userID, sessionTokenHash(loginCode), time.Now().UTC().Add(2*time.Minute))
	if err != nil {
		a.redirectOAuthError(w, r, "login_code_failed")
		return
	}
	target, err := url.Parse(a.Config.PublicURL)
	if err != nil || target.Scheme == "" || target.Host == "" {
		a.redirectOAuthError(w, r, "invalid_public_url")
		return
	}
	query := target.Query()
	query.Set("oauth_code", loginCode)
	query.Set("oauth_provider", "facebook")
	target.RawQuery = query.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

func facebookOAuthConfig() (string, string, string, string, error) {
	clientID := strings.TrimSpace(os.Getenv("FACEBOOK_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("FACEBOOK_CLIENT_SECRET"))
	redirectURI := strings.TrimSpace(os.Getenv("FACEBOOK_REDIRECT_URI"))
	version := strings.TrimSpace(os.Getenv("FACEBOOK_GRAPH_API_VERSION"))
	if version == "" {
		version = "v23.0"
	}
	if clientID == "" || clientSecret == "" || redirectURI == "" {
		return "", "", "", "", errors.New("facebook_oauth_not_configured")
	}
	parsed, err := url.Parse(redirectURI)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || !strings.HasPrefix(version, "v") {
		return "", "", "", "", errors.New("invalid_facebook_oauth_config")
	}
	return clientID, clientSecret, redirectURI, version, nil
}

func (a *App) exchangeFacebookCode(ctx context.Context, clientID, secret, redirectURI, version, code string) (string, error) {
	endpoint := "https://graph.facebook.com/" + version + "/oauth/access_token?" + url.Values{"client_id": {clientID}, "client_secret": {secret}, "redirect_uri": {redirectURI}, "code": {code}}.Encode()
	var decoded struct {
		AccessToken string `json:"access_token"`
	}
	if err := a.facebookJSON(ctx, endpoint, &decoded); err != nil || decoded.AccessToken == "" {
		return "", errors.New("facebook token response missing access_token")
	}
	return decoded.AccessToken, nil
}

func (a *App) validateFacebookToken(ctx context.Context, clientID, secret, version, token string) error {
	endpoint := "https://graph.facebook.com/" + version + "/debug_token?" + url.Values{"input_token": {token}, "access_token": {clientID + "|" + secret}}.Encode()
	var decoded struct {
		Data struct {
			AppID     string `json:"app_id"`
			IsValid   bool   `json:"is_valid"`
			UserID    string `json:"user_id"`
			ExpiresAt int64  `json:"expires_at"`
		} `json:"data"`
	}
	if err := a.facebookJSON(ctx, endpoint, &decoded); err != nil {
		return err
	}
	if !decoded.Data.IsValid || decoded.Data.AppID != clientID || decoded.Data.UserID == "" || (decoded.Data.ExpiresAt > 0 && decoded.Data.ExpiresAt <= time.Now().Unix()) {
		return errors.New("invalid facebook access token")
	}
	return nil
}

func (a *App) fetchFacebookProfile(ctx context.Context, version, token string) (facebookProfile, error) {
	endpoint := "https://graph.facebook.com/" + version + "/me?" + url.Values{"fields": {"id,name,email,picture.type(large)"}, "access_token": {token}}.Encode()
	var profile facebookProfile
	err := a.facebookJSON(ctx, endpoint, &profile)
	profile.Email = strings.ToLower(strings.TrimSpace(profile.Email))
	profile.Name = strings.TrimSpace(profile.Name)
	if err != nil || profile.ID == "" || profile.Email == "" {
		return facebookProfile{}, errors.New("facebook account must provide an email address")
	}
	if profile.Name == "" {
		profile.Name = strings.Split(profile.Email, "@")[0]
	}
	return profile, nil
}

func (a *App) facebookJSON(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := a.oauthHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("facebook graph returned %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(target)
}

func (a *App) linkFacebookIdentity(ctx context.Context, profile facebookProfile) (string, error) {
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var userID string
	err = tx.QueryRowContext(ctx, `select user_id from sys_oauth_accounts where provider='facebook' and provider_account_id=$1 for update`, profile.ID).Scan(&userID)
	if err == nil {
		_, err = tx.ExecContext(ctx, `update sys_oauth_accounts set email=$1,display_name=$2,avatar_url=$3,last_login_at=current_timestamp where provider='facebook' and provider_account_id=$4`, profile.Email, profile.Name, nullIfEmpty(profile.Picture.Data.URL), profile.ID)
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
		orgID := "org_" + randomHex(8)
		projectID := "project_" + randomHex(8)
		if _, err = tx.ExecContext(ctx, `insert into sys_users(id,email,name,avatar_url,status,password_hash,user_type,ui_theme,language) values($1,$2,$3,$4,'active',null,'individual_consumer','Light','EN')`, userID, profile.Email, profile.Name, nullIfEmpty(profile.Picture.Data.URL)); err != nil {
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
	_, err = tx.ExecContext(ctx, `insert into sys_oauth_accounts(id,user_id,provider,provider_account_id,email,display_name,avatar_url,last_login_at) values($1,$2,'facebook',$3,$4,$5,$6,current_timestamp)`, "oauth_"+randomHex(8), userID, profile.ID, profile.Email, profile.Name, nullIfEmpty(profile.Picture.Data.URL))
	if err != nil {
		return "", err
	}
	return userID, tx.Commit()
}

func setFacebookOAuthCookie(w http.ResponseWriter, value string, secure bool, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: facebookStateCookie, Value: value, Path: "/api/v1/auth/oauth/facebook", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
}
