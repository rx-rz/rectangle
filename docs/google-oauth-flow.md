# Google OAuth Flow

This is a "read this code" guide for the Google OAuth path across the web app and API.

The current shape is backend-owned:

```text
Frontend button
  -> API /auth/google/start
  -> Google
  -> API /auth/google/callback
  -> Frontend /
```

Google redirects back to the Go API, not to TanStack Router. The API owns the OAuth cookies, Google code exchange, local user lookup/create, local session creation, and final redirect back to the web app.

## Main Files

- `apps/web/src/features/auth/signup/containers/oauth.tsx`
  Starts the flow from the Google button.

- `apps/web/src/api/auth/get-google-oauth-link.ts`
  Calls `GET /auth/google/start`.

- `apps/api/internal/auth/handler.go`
  Contains `StartGoogleOauth`, `HandleGoogleOauth`, `exchangeGoogleCode`, and `fetchGoogleUser`.

- `apps/api/internal/auth/service.go`
  Contains `LoginWithGoogle`.

- `apps/api/internal/auth/repository.go`
  Contains the transaction: `FindOrCreateOAuthUserWithSession`.

- `apps/api/internal/config/config.go`
  Loads `GOOGLE_REDIRECT_URI` and `WEB_APP_URL`.

## Step 1: User Clicks Google

The button lives in:

```text
apps/web/src/features/auth/signup/containers/oauth.tsx
```

On click, it runs:

```tsx
const result = await getGoogleOauthLink();
const authUrl = result.data?.authUrl;
if (authUrl) {
  window.location.assign(authUrl);
}
```

The frontend does not build the Google URL. It asks the API for it.

The API wrapper is:

```text
apps/web/src/api/auth/get-google-oauth-link.ts
```

It calls:

```http
GET /auth/google/start
```

## Step 2: API Starts OAuth

The route is registered in:

```text
apps/api/internal/server/server.go
```

```go
opts.Mux.HandleFunc("GET /auth/google/start", opts.AuthHandler.StartGoogleOauth)
```

The handler is in:

```text
apps/api/internal/auth/handler.go
```

```go
func (h *Handler) StartGoogleOauth(w http.ResponseWriter, r *http.Request)
```

It creates two security values:

```go
state := rand.Text()
codeVerifier, codeChallenge := generatePKCE()
```

`state` protects against OAuth CSRF.

`codeVerifier` and `codeChallenge` are for PKCE. The challenge goes to Google. The verifier stays private in an HttpOnly cookie and is used later when the API exchanges Google's code.

The API stores both values in cookies:

```go
oauth_state
oauth_code_verifier
```

Both are `HttpOnly`, so frontend JavaScript cannot read them. The browser carries them back to the API callback automatically.

Then the handler builds the Google authorization URL:

```go
authUrl := buildGoogleAuthURL(...)
```

The Google URL includes:

```text
client_id
redirect_uri
response_type=code
scope=openid email profile
state
code_challenge
code_challenge_method=S256
prompt=select_account
```

The important part for this fix is `redirect_uri`. It should point to the Go API callback:

```text
http://localhost:4001/auth/google/callback
```

The API returns:

```json
{
  "authUrl": "https://accounts.google.com/o/oauth2/v2/auth?..."
}
```

Then the frontend redirects the browser to that URL.

## Step 3: Google Redirects To API

After the user chooses a Google account, Google redirects the browser to:

```text
GOOGLE_REDIRECT_URI
```

For local development, that should be:

```text
http://localhost:4001/auth/google/callback
```

Google adds query params:

```text
/auth/google/callback?code=...&state=...
```

This goes directly to Go:

```go
opts.Mux.HandleFunc("GET /auth/google/callback", opts.AuthHandler.HandleGoogleOauth)
```

## Step 4: API Handles Callback

The handler is:

```go
func (h *Handler) HandleGoogleOauth(w http.ResponseWriter, r *http.Request)
```

It reads Google's callback values:

```go
code := r.URL.Query().Get("code")
state := r.URL.Query().Get("state")
```

Then it reads the cookies created during `/auth/google/start`:

```go
savedState, err := helpers.ReadCookie(r, "oauth_state")
codeVerifier, err := helpers.ReadCookie(r, "oauth_code_verifier")
```

Then it checks:

```go
if state != savedState {
  redirectOAuthError(...)
  return
}
```

If this fails, the OAuth callback does not match the flow the API started.

## Step 5: API Exchanges Code With Google

The handler calls:

```go
tokenResp, err := exchangeGoogleCode(...)
```

`exchangeGoogleCode` makes a server-to-server request to:

```text
https://oauth2.googleapis.com/token
```

It sends:

```text
code
client_id
client_secret
redirect_uri
grant_type=authorization_code
code_verifier
```

The `code_verifier` proves the API owns the original PKCE verifier from the start step.

Google returns an access token.

## Step 6: API Fetches Google User

The handler calls:

```go
googleUser, err := fetchGoogleUser(r.Context(), tokenResp.AccessToken)
```

That calls:

```text
https://openidconnect.googleapis.com/v1/userinfo
```

The response is decoded into:

```go
type GoogleUser struct {
  ID            string `json:"sub"`
  Email         string `json:"email"`
  EmailVerified bool   `json:"email_verified"`
  Name          string `json:"name"`
  Picture       string `json:"picture"`
}
```

The most important field is `ID`. It is Google's stable user ID for this app.

## Step 7: Service Creates Local Login

The handler calls:

```go
result, err := h.service.LoginWithGoogle(...)
```

The service lives in:

```text
apps/api/internal/auth/service.go
```

It rejects unverified Google emails:

```go
if !input.EmailVerified {
  return nil, apperror.Unauthorized("google email is not verified")
}
```

Then it creates a local session token:

```go
token, tokenHash, err := newSessionToken()
```

The raw token is returned to the handler. Only the hash is stored in the database:

```go
hash := sha256.Sum256([]byte(token))
```

Then the service calls the repository transaction:

```go
result, err := s.oauthRepo.FindOrCreateOAuthUserWithSession(...)
```

## Step 8: Repository Transaction

The transaction lives in:

```text
apps/api/internal/auth/repository.go
```

The method is:

```go
func (r *Repository) FindOrCreateOAuthUserWithSession(...)
```

It does all database changes in one transaction.

First it looks up by Google identity:

```go
dbUser, err := findOAuthUser(ctx, tx, params.Provider, params.ProviderUserID)
```

The lookup is by:

```text
(provider, provider_user_id)
```

not by email.

If a linked OAuth account exists, it returns that local user.

If none exists, it creates a new local user:

```go
dbUser, err = createOAuthUser(ctx, tx, params)
```

That is a plain insert, not an upsert:

```sql
INSERT INTO users (id, name, email, avatar_url, email_verified_at)
VALUES ($1, $2, $3, $4, now())
```

Then it links the Google account:

```go
linkOAuthAccount(ctx, tx, params.Provider, params.ProviderUserID, dbUser.ID)
```

The link insert is:

```sql
INSERT INTO oauth_accounts (provider, provider_user_id, user_id)
VALUES ($1, $2, $3)
ON CONFLICT (provider, provider_user_id) DO NOTHING
```

If the insert affects zero rows, that means a conflict/race happened. The repo returns `ErrOAuthAccountLinked`.

Finally it creates a session:

```go
session, err := createSession(ctx, tx, params, dbUser.ID)
```

The session stores:

```text
session id
user id
user agent
token hash
ip address
expiry
```

Then the transaction commits.

## Step 9: API Sets Session Cookie And Redirects

After `LoginWithGoogle` succeeds, the callback handler sets the app session cookie:

```go
setSessionCookie(w, result.Token, result.Session.ExpiresAt, h.service.cfg.AppEnv == "production")
```

The cookie is:

```text
session_token
```

It is `HttpOnly`, so frontend JavaScript cannot read it.

Then the handler clears the temporary OAuth cookies:

```go
clearCookie(w, "oauth_state")
clearCookie(w, "oauth_code_verifier")
```

Finally it redirects the browser to the frontend:

```go
http.Redirect(w, r, h.service.cfg.WebAppURL, http.StatusFound)
```

For local development, that should land at:

```text
http://localhost:3000
```

## Mental Model

There are two identities:

1. Google identity:
   `provider = google`, `provider_user_id = Google sub`

2. Rectangle identity:
   `users.id = user_...`

`oauth_accounts` connects them.

The rule is:

```text
Google ID known     -> return linked Rectangle user and create a session
Google ID unknown   -> create Rectangle user, link Google ID, create session
Google ID conflict  -> reject
```

No email upsert is involved.

## Config Checklist

For local API env:

```text
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GOOGLE_REDIRECT_URI=http://localhost:4001/auth/google/callback
WEB_APP_URL=http://localhost:3000
```

For local frontend env:

```text
VITE_API_URL=http://localhost:4001
```

The Google Cloud Console authorized redirect URI must exactly match:

```text
http://localhost:4001/auth/google/callback
```

## Common Things To Check

If the callback ends at `/auth/login?oauth_error=invalid_oauth_state`, check:

- Did the user start at `/auth/google/start` first?
- Is Google redirecting to the same API host that set the cookies?
- Is `GOOGLE_REDIRECT_URI` exactly the API callback URL?
- Are cookies being blocked by browser settings?

If Google says `redirect_uri_mismatch`, check:

- Google Cloud Console authorized redirect URI.
- `GOOGLE_REDIRECT_URI` in the API environment.
- Use the API callback, not the frontend callback.

If user creation fails with a duplicate email:

- The Google ID was unknown, but the email already exists in `users`.
- The current implementation does not silently merge by email.
- A future "link Google to existing logged-in user" flow should be separate from normal Google login.
