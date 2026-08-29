# Google and Facebook Authentication Integration

Model Market currently includes a development-only social login placeholder.
Real Google and Facebook login requires OAuth applications, provider
credentials, and backend authorization-code callback handlers.

Basic login testing does not normally require a paid cloud account. You do
need developer applications with Google and Meta, and Meta requires a Facebook
account registered as a developer.

## Callback URLs

Use these backend callback URLs for local development:

```text
http://localhost:8080/api/v1/auth/oauth/google/callback
http://localhost:8080/api/v1/auth/oauth/facebook/callback
```

Production callbacks must use the deployed HTTPS backend domain, for example:

```text
https://api.example.com/api/v1/auth/oauth/google/callback
https://api.example.com/api/v1/auth/oauth/facebook/callback
```

OAuth redirect URLs must match the URLs configured at the provider exactly,
including the scheme, hostname, port, path, case, and trailing slash.

## Google Setup

1. Open the [Google Cloud Console](https://console.cloud.google.com/).
2. Create or select a Google Cloud project.
3. Configure the OAuth consent screen.
4. Select an external application unless access should be limited to one
   Google Workspace organization.
5. Add the developers' Google accounts as test users.
6. Open the credentials page and create an OAuth client.
7. Select **Web application** as the application type.
8. Add this authorized redirect URI for local development:

   ```text
   http://localhost:8080/api/v1/auth/oauth/google/callback
   ```

9. Store the generated client ID and client secret in backend environment
   variables:

   ```env
   GOOGLE_CLIENT_ID=
   GOOGLE_CLIENT_SECRET=
   GOOGLE_REDIRECT_URI=http://localhost:8080/api/v1/auth/oauth/google/callback
   ```

10. Request only the identity scopes needed for login:

    ```text
    openid email profile
    ```

Google permits localhost HTTP callbacks for development. Production callbacks
must use HTTPS. A public application will also need an authorized and verified
domain, an application homepage, and a privacy policy. Google may require
consent-screen verification depending on the requested scopes and audience.

References:

- [OAuth 2.0 for web-server applications](https://developers.google.com/identity/protocols/oauth2/web-server)
- [Google OAuth production readiness](https://developers.google.com/identity/protocols/oauth2/production-readiness/policy-compliance)

## Facebook Setup

1. Sign in at [Meta for Developers](https://developers.facebook.com/) with a
   Facebook account.
2. Complete Meta developer registration if prompted.
3. Create a Meta application appropriate for consumer authentication.
4. Add the Facebook Login product to the application.
5. Configure this valid OAuth redirect URI for local development:

   ```text
   http://localhost:8080/api/v1/auth/oauth/facebook/callback
   ```

6. Add the developers' Facebook accounts as administrators, developers, or
   testers while the application is in development mode.
7. Store the Meta application ID and secret in backend environment variables:

   ```env
   FACEBOOK_CLIENT_ID=
   FACEBOOK_CLIENT_SECRET=
   FACEBOOK_REDIRECT_URI=http://localhost:8080/api/v1/auth/oauth/facebook/callback
   ```

8. Request only the basic login permissions:

   ```text
   public_profile email
   ```

While the Meta application is in development mode, login is generally limited
to application administrators, developers, testers, and test users. Before a
public launch, switch the application to live mode and complete Meta's current
domain, privacy-policy, data-deletion, verification, and review requirements.

If Meta does not accept a localhost callback for the selected application
configuration, expose the local backend through an HTTPS development tunnel
and register that exact callback, for example:

```text
https://your-development-domain.example/api/v1/auth/oauth/facebook/callback
```

Reference:

- [Facebook Login for Web](https://developers.facebook.com/docs/facebook-login/web/)

## Backend Endpoints to Implement

The backend should expose these routes:

```text
GET /api/v1/auth/oauth/google/start
GET /api/v1/auth/oauth/google/callback
GET /api/v1/auth/oauth/facebook/start
GET /api/v1/auth/oauth/facebook/callback
```

The flow for each provider is:

1. The frontend sends the browser to the provider's `start` endpoint.
2. The backend generates a cryptographically random, short-lived `state`
   value and stores only the information needed to validate it.
3. The backend redirects the browser to the provider's authorization page.
4. The provider returns an authorization code to the registered callback.
5. The backend verifies that the returned `state` matches the stored value.
6. The backend exchanges the one-time authorization code for provider tokens.
7. The backend retrieves and verifies the provider account ID and email.
8. The backend creates or safely links `sys_users` and `sys_oauth_accounts`.
9. The backend creates a Model Market session in `sys_sessions` using the
   existing persistent session implementation.
10. The backend redirects the browser to the frontend without leaving the
    authorization code or provider tokens in the URL.

## Account-Linking Rules

- Treat the provider's stable account ID as the primary external identity.
- Link by email only when the provider states that the email is verified.
- Do not silently replace an existing provider link.
- Require an authenticated confirmation flow when linking another provider to
  an existing account.
- Prevent one provider account from being linked to multiple Model Market
  users.
- Record login and account-linking events in the audit log.

## Security Requirements

- Generate a fresh, unpredictable `state` value for every login attempt.
- Expire and consume each `state` value after one use.
- Use OpenID Connect nonce validation for Google ID tokens.
- Verify Google ID-token signature, issuer, audience, expiration, and nonce.
- Retrieve or validate Facebook identity using Meta's supported server-side
  endpoints; do not trust identity values supplied by the browser.
- Keep client secrets and provider tokens out of frontend code, URLs, logs,
  source control, and API responses.
- Store secrets in backend environment variables for development and in a
  managed secret store for production.
- Request the smallest possible set of scopes.
- Use HTTPS for every production authorization and callback URL.
- Redirect away from the callback immediately after processing it so the
  authorization code is not exposed to page resources or analytics scripts.
- Apply login throttling and structured security logging to OAuth endpoints.

## Local Environment Example

Add these values to a local `.env` file that is excluded from source control:

```env
PUBLIC_URL=http://localhost:3000

GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_REDIRECT_URI=http://localhost:8080/api/v1/auth/oauth/google/callback

FACEBOOK_CLIENT_ID=
FACEBOOK_CLIENT_SECRET=
FACEBOOK_REDIRECT_URI=http://localhost:8080/api/v1/auth/oauth/facebook/callback
```

Do not commit real values. The repository's `.env.example` should contain only
empty placeholders and safe example callback URLs.

## Production Checklist

- Replace localhost callbacks with the production HTTPS backend domain.
- Configure separate OAuth applications or credentials for development,
  staging, and production.
- Verify all production domains with Google and Meta.
- Publish a homepage, privacy policy, terms, and data-deletion instructions.
- Keep development/test accounts separate from production administrators.
- Confirm that only intended test users can access applications in test mode.
- Complete provider verification or application review where required.
- Rotate any credential accidentally exposed in source control or logs.
- Test successful login, denial, invalid state, expired state, replayed
  callback, missing email, unverified email, account conflicts, logout, and
  expired Model Market sessions.
