package httpapi

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	googleAuthorizationEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenEndpoint         = "https://oauth2.googleapis.com/token"
	googleJWKSEndpoint          = "https://www.googleapis.com/oauth2/v3/certs"
	googleStateCookie           = "mm_google_oauth_state"
	googleNonceCookie           = "mm_google_oauth_nonce"
	googleVerifierCookie        = "mm_google_oauth_verifier"
)

type googleTokenResponse struct {
	IDToken string `json:"id_token"`
}

type googleIDClaims struct {
	Issuer        string `json:"iss"`
	Subject       string `json:"sub"`
	Audience      string `json:"aud"`
	AuthorizedBy  string `json:"azp"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Nonce         string `json:"nonce"`
	ExpiresAt     int64  `json:"exp"`
	IssuedAt      int64  `json:"iat"`
}

func (a *App) googleOAuthStart(w http.ResponseWriter, r *http.Request) {
	clientID, _, redirectURI, err := googleOAuthConfig()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, response{"error": err.Error()})
		return
	}
	state := randomHex(32)
	nonce := randomHex(32)
	verifier := randomHex(32)
	challengeRaw := sha256.Sum256([]byte(verifier))
	secure := strings.HasPrefix(strings.ToLower(redirectURI), "https://")
	setOAuthCookie(w, googleStateCookie, state, secure, 600)
	setOAuthCookie(w, googleNonceCookie, nonce, secure, 600)
	setOAuthCookie(w, googleVerifierCookie, verifier, secure, 600)
	query := url.Values{
		"client_id":              {clientID},
		"redirect_uri":           {redirectURI},
		"response_type":          {"code"},
		"scope":                  {"openid email profile"},
		"state":                  {state},
		"nonce":                  {nonce},
		"code_challenge":         {base64.RawURLEncoding.EncodeToString(challengeRaw[:])},
		"code_challenge_method":  {"S256"},
		"include_granted_scopes": {"true"},
		"prompt":                 {"select_account"},
	}
	http.Redirect(w, r, googleAuthorizationEndpoint+"?"+query.Encode(), http.StatusFound)
}

func (a *App) googleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	clientID, clientSecret, redirectURI, err := googleOAuthConfig()
	if err != nil {
		a.redirectOAuthError(w, r, "google_not_configured")
		return
	}
	secure := strings.HasPrefix(strings.ToLower(redirectURI), "https://")
	defer clearGoogleOAuthCookies(w, secure)
	if providerError := strings.TrimSpace(r.URL.Query().Get("error")); providerError != "" {
		a.redirectOAuthError(w, r, providerError)
		return
	}
	if issuer := strings.TrimSpace(r.URL.Query().Get("iss")); issuer != "" && issuer != "https://accounts.google.com" {
		a.redirectOAuthError(w, r, "invalid_issuer")
		return
	}
	stateCookie, stateErr := r.Cookie(googleStateCookie)
	nonceCookie, nonceErr := r.Cookie(googleNonceCookie)
	verifierCookie, verifierErr := r.Cookie(googleVerifierCookie)
	returnedState := r.URL.Query().Get("state")
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if stateErr != nil || nonceErr != nil || verifierErr != nil || returnedState == "" || code == "" || subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(returnedState)) != 1 {
		a.redirectOAuthError(w, r, "invalid_oauth_state")
		return
	}
	token, err := a.exchangeGoogleCode(r.Context(), clientID, clientSecret, redirectURI, code, verifierCookie.Value)
	if err != nil {
		a.redirectOAuthError(w, r, "token_exchange_failed")
		return
	}
	claims, err := a.verifyGoogleIDToken(r.Context(), token.IDToken, clientID, nonceCookie.Value)
	if err != nil {
		a.redirectOAuthError(w, r, "invalid_identity_token")
		return
	}
	userID, err := a.linkGoogleIdentity(r.Context(), claims)
	if err != nil {
		a.redirectOAuthError(w, r, "account_link_failed")
		return
	}
	loginCode := "oauth_" + randomHex(32)
	_, err = a.DB.ExecContext(r.Context(), `insert into sys_oauth_login_codes(id,user_id,code_hash,provider,expires_at) values($1,$2,$3,'google',$4)`, "oauth_code_"+randomHex(12), userID, sessionTokenHash(loginCode), time.Now().UTC().Add(2*time.Minute))
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
	query.Set("oauth_provider", "google")
	target.RawQuery = query.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

func (a *App) oauthLoginExchange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req) != nil || strings.TrimSpace(req.Code) == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_login_code"})
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "login_exchange_failed"})
		return
	}
	defer tx.Rollback()
	var userID, email string
	err = tx.QueryRowContext(r.Context(), `select c.user_id,u.email from sys_oauth_login_codes c join sys_users u on u.id = c.user_id where c.code_hash = $1 and c.expires_at > current_timestamp for update`, sessionTokenHash(strings.TrimSpace(req.Code))).Scan(&userID, &email)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, response{"error": "invalid_or_expired_login_code"})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `delete from sys_oauth_login_codes where code_hash = $1`, sessionTokenHash(strings.TrimSpace(req.Code))); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "login_exchange_failed"})
		return
	}
	if err = tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "login_exchange_failed"})
		return
	}
	login, err := a.loginByEmail(r.Context(), email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "login_exchange_failed"})
		return
	}
	login["provider"] = "google"
	writeJSON(w, http.StatusOK, login)
}

func googleOAuthConfig() (string, string, string, error) {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET"))
	redirectURI := strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URI"))
	if clientID == "" || clientSecret == "" || redirectURI == "" {
		return "", "", "", errors.New("google_oauth_not_configured")
	}
	parsed, err := url.Parse(redirectURI)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", "", errors.New("invalid_google_redirect_uri")
	}
	return clientID, clientSecret, redirectURI, nil
}

func (a *App) exchangeGoogleCode(ctx context.Context, clientID, clientSecret, redirectURI, code, verifier string) (googleTokenResponse, error) {
	form := url.Values{"client_id": {clientID}, "client_secret": {clientSecret}, "code": {code}, "code_verifier": {verifier}, "grant_type": {"authorization_code"}, "redirect_uri": {redirectURI}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return googleTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.oauthHTTPClient().Do(req)
	if err != nil {
		return googleTokenResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return googleTokenResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return googleTokenResponse{}, fmt.Errorf("google token endpoint returned %d", resp.StatusCode)
	}
	var token googleTokenResponse
	if json.Unmarshal(body, &token) != nil || token.IDToken == "" {
		return googleTokenResponse{}, errors.New("google token response missing id_token")
	}
	return token, nil
}

func (a *App) verifyGoogleIDToken(ctx context.Context, rawToken, clientID, expectedNonce string) (googleIDClaims, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return googleIDClaims{}, errors.New("malformed id token")
	}
	decode := func(value string, target any) error {
		raw, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, target)
	}
	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	var claims googleIDClaims
	if decode(parts[0], &header) != nil || decode(parts[1], &claims) != nil || header.Algorithm != "RS256" || header.KeyID == "" {
		return googleIDClaims{}, errors.New("invalid id token header")
	}
	key, err := a.googlePublicKey(ctx, header.KeyID)
	if err != nil {
		return googleIDClaims{}, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return googleIDClaims{}, err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature) != nil {
		return googleIDClaims{}, errors.New("invalid id token signature")
	}
	now := time.Now().Unix()
	validIssuer := claims.Issuer == "https://accounts.google.com" || claims.Issuer == "accounts.google.com"
	if !validIssuer || claims.Audience != clientID || (claims.AuthorizedBy != "" && claims.AuthorizedBy != clientID) || claims.ExpiresAt <= now || claims.IssuedAt > now+300 || claims.Subject == "" || !claims.EmailVerified || strings.TrimSpace(claims.Email) == "" || subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(expectedNonce)) != 1 {
		return googleIDClaims{}, errors.New("invalid id token claims")
	}
	claims.Email = strings.ToLower(strings.TrimSpace(claims.Email))
	claims.Name = strings.TrimSpace(claims.Name)
	if claims.Name == "" {
		claims.Name = strings.Split(claims.Email, "@")[0]
	}
	return claims, nil
}

func (a *App) googlePublicKey(ctx context.Context, keyID string) (*rsa.PublicKey, error) {
	a.googleKeysMu.Lock()
	defer a.googleKeysMu.Unlock()
	if time.Now().Before(a.googleKeysExpires) {
		if key := a.googleKeys[keyID]; key != nil {
			return key, nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleJWKSEndpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.oauthHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google jwks returned %d", resp.StatusCode)
	}
	var keys struct {
		Keys []struct {
			KeyID     string `json:"kid"`
			KeyType   string `json:"kty"`
			Algorithm string `json:"alg"`
			Modulus   string `json:"n"`
			Exponent  string `json:"e"`
		} `json:"keys"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&keys) != nil {
		return nil, errors.New("invalid google jwks")
	}
	parsedKeys := map[string]*rsa.PublicKey{}
	for _, item := range keys.Keys {
		if item.KeyID == "" || item.KeyType != "RSA" || item.Algorithm != "RS256" {
			continue
		}
		nBytes, nErr := base64.RawURLEncoding.DecodeString(item.Modulus)
		eBytes, eErr := base64.RawURLEncoding.DecodeString(item.Exponent)
		if nErr != nil || eErr != nil || len(nBytes) == 0 || len(eBytes) == 0 {
			return nil, errors.New("invalid google signing key")
		}
		exponent := 0
		for _, value := range eBytes {
			exponent = exponent<<8 | int(value)
		}
		if exponent < 3 {
			return nil, errors.New("invalid google signing exponent")
		}
		parsedKeys[item.KeyID] = &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: exponent}
	}
	maxAge := 3600
	for _, directive := range strings.Split(resp.Header.Get("Cache-Control"), ",") {
		name, value, found := strings.Cut(strings.TrimSpace(directive), "=")
		if found && strings.EqualFold(name, "max-age") {
			if seconds, parseErr := strconv.Atoi(strings.TrimSpace(value)); parseErr == nil && seconds > 0 {
				maxAge = seconds
			}
		}
	}
	a.googleKeys = parsedKeys
	a.googleKeysExpires = time.Now().Add(time.Duration(maxAge) * time.Second)
	if key := parsedKeys[keyID]; key != nil {
		return key, nil
	}
	return nil, errors.New("google signing key not found")
}

func (a *App) linkGoogleIdentity(ctx context.Context, claims googleIDClaims) (string, error) {
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var userID string
	err = tx.QueryRowContext(ctx, `select user_id from sys_oauth_accounts where provider = 'google' and provider_account_id = $1 for update`, claims.Subject).Scan(&userID)
	if err == nil {
		_, err = tx.ExecContext(ctx, `update sys_oauth_accounts set email=$1,display_name=$2,avatar_url=$3,last_login_at=current_timestamp where provider='google' and provider_account_id=$4`, claims.Email, claims.Name, claims.Picture, claims.Subject)
		if err != nil {
			return "", err
		}
		return userID, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	err = tx.QueryRowContext(ctx, `select id from sys_users where lower(email)=lower($1) for update`, claims.Email).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		userID = "user_" + randomHex(8)
		orgID := "org_" + randomHex(8)
		projectID := "project_" + randomHex(8)
		if _, err = tx.ExecContext(ctx, `insert into sys_users(id,email,name,avatar_url,status,password_hash,user_type,ui_theme,language) values($1,$2,$3,$4,'active',null,'individual_consumer','Light','EN')`, userID, claims.Email, claims.Name, nullIfEmpty(claims.Picture)); err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `insert into sys_organizations(id,name,slug,status) values($1,$2,$3,'active')`, orgID, claims.Name+" Workspace", "personal-"+randomHex(8)); err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `insert into sys_memberships(id,user_id,organization_id,role) values($1,$2,$3,'owner')`, "membership_"+randomHex(8), userID, orgID); err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `insert into user_projects(id,organization_id,name,slug,environment,retention_policy) values($1,$2,'My Project',$3,'dev','{"conversation_days":30,"asset_days":30}')`, projectID, orgID, "my-project-"+randomHex(6)); err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `insert into user_wallets(id,project_id,paid_credits,promotional_credits) values($1,$2,0,1000)`, "wallet_"+randomHex(8), projectID); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	_, err = tx.ExecContext(ctx, `insert into sys_oauth_accounts(id,user_id,provider,provider_account_id,email,display_name,avatar_url,last_login_at) values($1,$2,'google',$3,$4,$5,$6,current_timestamp)`, "oauth_"+randomHex(8), userID, claims.Subject, claims.Email, claims.Name, nullIfEmpty(claims.Picture))
	if err != nil {
		return "", err
	}
	return userID, tx.Commit()
}

func (a *App) redirectOAuthError(w http.ResponseWriter, r *http.Request, code string) {
	target, err := url.Parse(a.Config.PublicURL)
	if err != nil || target.Scheme == "" || target.Host == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": code})
		return
	}
	query := target.Query()
	query.Set("oauth_error", code)
	target.RawQuery = query.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

func (a *App) oauthHTTPClient() *http.Client {
	if a.Client != nil {
		return a.Client
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func setOAuthCookie(w http.ResponseWriter, name, value string, secure bool, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/api/v1/auth/oauth/google", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
}

func clearGoogleOAuthCookies(w http.ResponseWriter, secure bool) {
	for _, name := range []string{googleStateCookie, googleNonceCookie, googleVerifierCookie} {
		setOAuthCookie(w, name, "", secure, -1)
	}
}
