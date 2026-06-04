# Rectangle API Walkthrough For A Hono Brain

This file explains the Go backend as if you already understand a Node backend built with Hono.

The goal is not just "what does this code do?" The goal is "what mental model should I use while writing more of this API?"

At a high level, this API is doing the same things a small Hono app would do:

```ts
const app = new Hono()

const db = createDb(process.env.DATABASE_URL)
const userRepo = new UserRepository(db)
const authRepo = new AuthRepository(db)
const authService = new AuthService({ userRepo, authRepo, logger })
const authHandler = new AuthHandler({ authService, logger })

app.get('/health', healthHandler)
app.post('/auth/signup/email', authHandler.signupWithEmail)

serve(app)
```

The Go version is more explicit because Go does not hide much behind framework magic. There is no global `app`, no decorators, no implicit dependency injection container, and no request context object shaped like Hono's `c`. Instead, you wire dependencies manually and each HTTP handler receives:

```go
func(w http.ResponseWriter, r *http.Request)
```

That pair is Go's lower-level version of Hono's `c`.

- `r *http.Request` is the incoming request: method, path, headers, body, context, etc.
- `w http.ResponseWriter` is how you write the response: status, headers, body.

The backend currently has one real route:

```txt
POST /auth/signup/email
```

And one health route:

```txt
GET /health
```

The signup request path is:

```txt
main.go
  loads config
  opens database
  creates repositories
  creates auth service
  creates HTTP server

server.New
  creates router/mux
  creates auth handler
  registers routes

auth.Handler.SignupWithEmail
  reads JSON
  normalizes input
  validates input
  calls auth service
  writes JSON response

auth.AuthService.SignupWithEmail
  checks if user exists
  hashes password
  creates user ID
  calls user repository

user.Repository.Create
  runs SQL INSERT
  scans database row into user.User model

handler response mapper
  converts database model into JSON DTO
```

Everything else in the codebase exists to support that path.

## Directory Map

The Go API lives under `apps/api`.

```txt
apps/api
  cmd/api/main.go
  internal/
    apperror/
    auth/
    config/
    db/
    helpers/
    server/
    user/
  migrations/
  platform/
    logger/
  go.mod
```

The rough meaning:

| Path | Hono-ish meaning |
| --- | --- |
| `cmd/api/main.go` | App bootstrap file. Similar to `src/index.ts` or `server.ts`. |
| `internal/server` | Router/server setup. Similar to where you create `new Hono()` and register routes. |
| `internal/auth` | Auth feature module: DTOs, handler, service, repository, models. |
| `internal/user` | User feature module: DB model, response DTO, repository, service placeholder. |
| `internal/helpers` | Shared utilities: JSON reading/writing, validation, hashing, IDs, OTPs. |
| `internal/apperror` | Shared application error type. Similar to a custom error class. |
| `internal/config` | Environment loading and validation. |
| `internal/db` | Database connection setup. |
| `platform/logger` | Logger creation. |
| `migrations` | SQL schema. |

The `internal` folder matters in Go. Code inside an `internal` directory can only be imported by code inside the parent tree. That means `rx-rz/rectangle-api/internal/auth` is private to this Go module. It is Go's built-in way of saying "this is app code, not a public library API."

## `main.go`: The Bootstrap File

File:

```txt
apps/api/cmd/api/main.go
```

This is the real entrypoint. In Go, the executable starts at:

```go
func main() {
}
```

Inside `main`, the API does this:

```go
cfg, err := config.Load()
```

This loads `.env` and environment variables into a typed `Config`.

Then:

```go
appLogger := logger.New(cfg.AppEnv)
```

This creates a `*slog.Logger`.

Then:

```go
database, err := db.Open(cfg.DbUrl)
```

This creates a database connection pool.

Then:

```go
userRepo := user.NewRepository(database)
authRepo := auth.NewRepository(database)
authService := auth.NewService(auth.ServiceOptions{
    UserRepository: userRepo,
    OTPRepository:  authRepo,
    Logger:         appLogger,
})
```

This is the part you asked about: why define `authService` with all of its stuff in `main.go`?

Because `main.go` is acting as the composition root.

In a Node/Hono app, you might do:

```ts
const db = createDb()
const userRepo = new UserRepository(db)
const authRepo = new AuthRepository(db)
const authService = new AuthService(userRepo, authRepo)

app.post('/auth/signup/email', async (c) => {
  return authService.signupWithEmail(c)
})
```

That setup has to happen somewhere. In this Go app, that "somewhere" is `main.go`.

### Why not create the auth service inside its own file?

You could make a helper like:

```go
func NewAuthService(db *sqlx.DB, logger *slog.Logger) *auth.AuthService {
    userRepo := user.NewRepository(db)
    authRepo := auth.NewRepository(db)
    return auth.NewService(auth.ServiceOptions{
        UserRepository: userRepo,
        OTPRepository: authRepo,
        Logger: logger,
    })
}
```

That is not wrong. But it changes where dependency wiring lives.

The current style keeps the dependency graph visible in one place:

```txt
database
  -> user repository
  -> auth repository
  -> auth service
  -> server
  -> route handlers
```

That is useful because you can see the whole backend's startup structure without chasing factory functions.

The key design idea is:

```txt
main.go owns object construction.
services own business logic.
repositories own SQL.
handlers own HTTP request/response.
```

That separation is very Go.

### Manual Dependency Injection

This code:

```go
authService := auth.NewService(auth.ServiceOptions{
    UserRepository: userRepo,
    OTPRepository:  authRepo,
    Logger:         appLogger,
})
```

is manual dependency injection.

No framework is injecting things for you. You are passing dependencies explicitly.

In Hono, you might use variables from closure scope:

```ts
const authService = new AuthService(...)

app.post('/signup', async (c) => {
  const result = await authService.signupWithEmail(...)
})
```

In this Go app, the equivalent is storing the service on a handler struct:

```go
type Handler struct {
    service *AuthService
    logger  *slog.Logger
}
```

Then each handler method can use:

```go
h.service.SignupWithEmail(...)
```

Same idea, different shape.

### Why Use Interfaces In `AuthService`?

In `auth/service.go`:

```go
type UserRepository interface {
    Create(ctx context.Context, params user.CreateUserParams) (*user.User, error)
    Update(ctx context.Context, params user.UpdateUserParams) (*user.User, error)
    FindByEmail(ctx context.Context, email string) (*user.User, error)
    GetPasswordHashByEmail(ctx context.Context, email string) (string, error)
}
```

The auth service does not require the concrete `user.Repository` type. It only requires "anything that has these methods."

In TypeScript terms:

```ts
type UserRepository = {
  create(params: CreateUserParams): Promise<User>
  update(params: UpdateUserParams): Promise<User>
  findByEmail(email: string): Promise<User | null>
  getPasswordHashByEmail(email: string): Promise<string>
}
```

This makes `AuthService` easier to test because you can pass a fake repository that implements the same methods.

In Go, you usually define the interface near the code that consumes it. That is why the `UserRepository` interface is in `auth/service.go`, not in `user/repository.go`.

Mental model:

```txt
The auth service says:
"I do not care what exact repository you give me.
I only care that it has the methods I need."
```

### Graceful Shutdown

This part:

```go
shutdownErr := make(chan error, 1)
go func() {
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    shutdownErr <- apiServer.Shutdown(ctx)
}()
```

is a graceful shutdown flow.

In Node, you might write:

```ts
process.on('SIGTERM', async () => {
  await server.close()
})
```

In Go:

- `go func() { ... }()` starts a lightweight concurrent function called a goroutine.
- `quit := make(chan os.Signal, 1)` creates a channel that receives OS signals.
- `signal.Notify(...)` tells Go to push interrupt/terminate signals into that channel.
- `<-quit` blocks until a signal arrives.
- `context.WithTimeout(..., 10*time.Second)` gives shutdown 10 seconds to finish.
- `apiServer.Shutdown(ctx)` stops accepting new requests and lets in-flight requests finish.

Then this starts the server:

```go
err = apiServer.ListenAndServe()
```

`ListenAndServe` blocks while the server is running. When shutdown happens normally, it returns `http.ErrServerClosed`, so this code ignores that expected error:

```go
if !errors.Is(err, http.ErrServerClosed) {
    appLogger.Error("api server failed", "error", err)
    os.Exit(1)
}
```

## `server.go`: Router And HTTP Server Setup

File:

```txt
apps/api/internal/server/server.go
```

This function creates the HTTP server:

```go
func New(opts Options) *http.Server {
    mux := http.NewServeMux()
    ...
    return &http.Server{...}
}
```

In Hono terms:

```ts
function createServer(opts) {
  const app = new Hono()
  registerRoutes(app, opts)
  return app
}
```

### `Options`

```go
type Options struct {
    Port        int
    AuthService *auth.AuthService
    Logger      *slog.Logger
}
```

This is a typed options object.

TypeScript equivalent:

```ts
type Options = {
  port: number
  authService: AuthService
  logger: Logger
}
```

Go has no object literal with inferred structural typing like TypeScript, so you define a `struct`.

### `http.NewServeMux`

```go
mux := http.NewServeMux()
```

`ServeMux` is Go's built-in router.

It maps HTTP route patterns to handlers.

This:

```go
opts.Mux.HandleFunc("GET /health", healthHandler)
opts.Mux.HandleFunc("POST /auth/signup/email", opts.AuthHandler.SignupWithEmail)
```

is similar to:

```ts
app.get('/health', healthHandler)
app.post('/auth/signup/email', authHandler.signupWithEmail)
```

The `"GET /health"` and `"POST /auth/signup/email"` style is available in modern Go's standard library router. Older Go examples often look like:

```go
mux.HandleFunc("/health", healthHandler)
```

and then manually check the method. This code is using the newer, cleaner pattern syntax.

### The Handler Object

```go
authHandler := auth.NewHandler(auth.HandlerOptions{
    Service: opts.AuthService,
    Logger:  opts.Logger,
})
```

This creates an object that knows how to handle auth HTTP requests.

In Node:

```ts
const authHandler = new AuthHandler({ service: authService, logger })
```

Then:

```go
opts.Mux.HandleFunc("POST /auth/signup/email", opts.AuthHandler.SignupWithEmail)
```

means:

```txt
When a POST request comes in for /auth/signup/email,
call the SignupWithEmail method on this handler instance.
```

### `http.Server` Timeout Settings

```go
return &http.Server{
    Addr:              fmt.Sprintf(":%d", opts.Port),
    Handler:           mux,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       10 * time.Second,
    WriteTimeout:      10 * time.Second,
    IdleTimeout:       60 * time.Second,
}
```

This is not just route setup. It creates the actual server with safety timeouts.

- `Addr`: what port to listen on.
- `Handler`: the router.
- `ReadHeaderTimeout`: max time to read request headers.
- `ReadTimeout`: max time to read the full request.
- `WriteTimeout`: max time to write the response.
- `IdleTimeout`: how long to keep idle keep-alive connections open.

In Node frameworks these may be hidden in the server implementation. In Go you often set them yourself.

## `auth/handler.go`: HTTP Layer

File:

```txt
apps/api/internal/auth/handler.go
```

This is the HTTP adapter for auth.

The handler is not where business logic should live. Its job is:

1. Read the request.
2. Normalize request data.
3. Validate request data.
4. Call the service.
5. Convert the service result into JSON.
6. Convert errors into HTTP responses.

That is exactly what `SignupWithEmail` does.

```go
func (h *Handler) SignupWithEmail(w http.ResponseWriter, r *http.Request) {
```

This is a method on `*Handler`.

The weird-looking part is:

```go
func (h *Handler) SignupWithEmail(...)
```

The `(h *Handler)` part is called the receiver.

TypeScript equivalent:

```ts
class Handler {
  signupWithEmail(c) {
    this.service.signupWithEmail(...)
  }
}
```

In Go, instead of writing `this`, you pick a receiver name. Here it is `h`.

So:

```go
h.service
```

means:

```ts
this.service
```

### Handler Dependencies

```go
type Handler struct {
    service *AuthService
    logger  *slog.Logger
}
```

This is like:

```ts
class AuthHandler {
  constructor(
    private service: AuthService,
    private logger: Logger,
  ) {}
}
```

And:

```go
type HandlerOptions struct {
    Service *AuthService
    Logger  *slog.Logger
}
```

is a constructor options object.

```go
func NewHandler(opts HandlerOptions) *Handler {
    return &Handler{
        service: opts.Service,
        logger:  opts.Logger,
    }
}
```

The `&Handler{...}` means "create a Handler value and return a pointer to it."

A pointer is roughly "a reference to this object" rather than a copy. In application code, Go services and handlers are commonly passed as pointers.

### Reading JSON

```go
var input EmailSignupInput
if err := helpers.ReadJSON(w, r, &input); err != nil {
    h.writeError(w, apperror.BadRequest(err.Error()))
    return
}
```

In Hono:

```ts
const input = await c.req.json<EmailSignupInput>()
```

In Go:

- `var input EmailSignupInput` creates an empty struct.
- `&input` passes a pointer so `ReadJSON` can mutate/fill it.
- If reading fails, return a bad request response.

Why pass `w` into `ReadJSON`? Because `ReadJSON` uses:

```go
r.Body = http.MaxBytesReader(w, r.Body, MAX_JSON_BODY_SIZE)
```

That caps the body size at 1 MB.

### Normalize Before Validation

```go
input = normalizeEmailSignupInput(input)
```

This trims and lowercases email:

```go
input.Email = strings.ToLower(strings.TrimSpace(input.Email))
```

It also cleans up optional name:

```go
if input.Name != nil {
    name := strings.TrimSpace(*input.Name)
    if name == "" {
        input.Name = nil
    } else {
        input.Name = &name
    }
}
```

Important Go concepts here:

- `Name *string` means name is optional.
- `nil` means absent/null/no value.
- `*input.Name` means "dereference the pointer and get the actual string."
- `&name` means "take the address of this string so it can be stored as `*string`."

In TypeScript:

```ts
type EmailSignupInput = {
  name?: string
  email: string
  password: string
}
```

The normalization logic is basically:

```ts
input.email = input.email.trim().toLowerCase()

if (input.name != null) {
  const name = input.name.trim()
  input.name = name === '' ? undefined : name
}
```

### Validate

```go
if err := helpers.ValidateStruct(input); err != nil {
    h.writeError(w, err)
    return
}
```

This uses validation tags from the DTO:

```go
type EmailSignupInput struct {
    Name     *string `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
    Email    string  `json:"email" validate:"required,email,max=255"`
    Password string  `json:"password" validate:"required,min=8,max=72"`
}
```

In Node this would be similar to Zod:

```ts
const EmailSignupInput = z.object({
  name: z.string().min(2).max(255).optional(),
  email: z.string().email().max(255),
  password: z.string().min(8).max(72),
})
```

The difference is that Go attaches metadata to struct fields using tags:

```go
`json:"email" validate:"required,email,max=255"`
```

The `json` tag tells the JSON encoder/decoder what field name to use.

The `validate` tag tells the validator library what rules to apply.

### Call The Service

```go
createdUser, err := h.service.SignupWithEmail(r.Context(), input)
```

`r.Context()` is important.

It gives the service a request-scoped context. If the client disconnects or the request times out, that context can be canceled. Database calls should receive that context so they can stop work.

In Node, this is kind of like passing an `AbortSignal` down to lower layers.

```ts
await authService.signupWithEmail({ input, signal: request.signal })
```

### Write Response

```go
err = helpers.WriteJSON(w, http.StatusCreated, helpers.Envelope{
    "data": AuthResponse{
        User: toAuthUserResponse(createdUser),
    },
}, nil)
```

In Hono:

```ts
return c.json({
  data: {
    user: toAuthUserResponse(createdUser),
  },
}, 201)
```

`helpers.Envelope` is:

```go
type Envelope map[string]any
```

That is a flexible object/map shape.

TypeScript equivalent:

```ts
type Envelope = Record<string, unknown>
```

`any` in Go is an alias for `interface{}`. It means "any type." This is one of the rare places where using `any` is fine because JSON envelopes can hold different shapes.

### Mapping Database Model To Response DTO

```go
func toAuthUserResponse(u *user.User) AuthUserResponse {
```

This converts the internal database model into the external JSON response shape.

The database model uses `sql.NullString`:

```go
Name sql.NullString
```

because SQL nullable columns are not plain strings. A plain Go `string` cannot represent SQL `NULL`; it can only represent an empty string.

So Go's SQL package uses:

```go
sql.NullString{
    String: "Temi",
    Valid: true,
}
```

or:

```go
sql.NullString{
    String: "",
    Valid: false,
}
```

This code:

```go
var name *string
if u.Name.Valid {
    name = &u.Name.String
}
```

means:

```txt
If the DB value was not NULL, expose a string pointer.
If it was NULL, leave name as nil.
```

Then the response DTO has:

```go
Name *string `json:"name,omitempty"`
```

`omitempty` means if `Name` is nil, omit it from the JSON response.

In TypeScript:

```ts
const name = row.name === null ? undefined : row.name
```

### Error Handling In The Handler

```go
func (h *Handler) writeError(w http.ResponseWriter, err error) {
    var appErr *apperror.Error
    if !errors.As(err, &appErr) {
        h.logger.Error("unhandled error", "error", err)
        appErr = apperror.Internal()
    }

    writeErr := helpers.WriteJSON(w, appErr.Status, helpers.Envelope{
        "error": helpers.Envelope{
            "code":    appErr.Code,
            "message": appErr.Message,
        },
    }, nil)
    ...
}
```

This is the equivalent of:

```ts
if (!(err instanceof AppError)) {
  logger.error(err)
  err = AppError.internal()
}

return c.json({
  error: {
    code: err.code,
    message: err.message,
  },
}, err.status)
```

`errors.As(err, &appErr)` checks whether the error is, or wraps, an `*apperror.Error`.

That lets lower layers return typed app errors like:

```go
apperror.Conflict("user already exists.")
```

and the handler knows what HTTP status to use.

## `auth/dto.go`: Request And Response Shapes

File:

```txt
apps/api/internal/auth/dto.go
```

DTO means Data Transfer Object. It is the shape of data crossing a boundary.

Here, the boundary is HTTP JSON.

```go
type EmailSignupInput struct {
    Name     *string `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
    Email    string  `json:"email" validate:"required,email,max=255"`
    Password string  `json:"password" validate:"required,min=8,max=72"`
}
```

This is not the database user. This is the request body for signup.

Compare:

```go
type AuthUserResponse struct {
    ID              string     `json:"id"`
    Name            *string    `json:"name,omitempty"`
    Email           string     `json:"email"`
    AvatarURL       *string    `json:"avatar_url,omitempty"`
    EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
    CreatedAt       time.Time  `json:"created_at"`
}
```

This is the response shape for a user inside auth responses.

The response intentionally does not include:

- `password_hash`
- internal SQL null wrappers
- update timestamps
- anything private

That is good. Keep DB models and API response DTOs separate.

## `helpers/validation.go`: The `reflect` Thing

File:

```txt
apps/api/internal/helpers/validation.go
```

This is the part that feels cursed if you are coming from JS:

```go
v.RegisterTagNameFunc(func(field reflect.StructField) string {
    name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
    if name == "-" {
        return ""
    }
    return name
})
```

Let's break it down.

### What Is Reflection?

Reflection is when code inspects types and values at runtime.

Normally, Go is very static:

```go
input.Email
```

You know at compile time that `input` has an `Email` field.

But a validation library receives this:

```go
validate.Struct(input)
```

It does not know your struct at compile time. It needs to inspect whatever struct you gave it:

- What fields does this struct have?
- What tags are on those fields?
- What values are inside those fields?
- Are the validation rules passing?

That is what reflection enables.

In JavaScript, this feels normal:

```ts
Object.keys(input)
```

or:

```ts
schema.shape.email
```

But in Go, types are not usually inspected dynamically unless you explicitly use the `reflect` package or a library uses it for you.

### What Is `reflect.StructField`?

`reflect.StructField` is metadata about one field in a struct.

For this struct:

```go
type EmailSignupInput struct {
    Email string `json:"email" validate:"required,email,max=255"`
}
```

reflection can inspect the `Email` field and produce metadata like:

```txt
field.Name = "Email"
field.Type = string
field.Tag  = `json:"email" validate:"required,email,max=255"`
```

The validator library calls your function for each struct field:

```go
func(field reflect.StructField) string {
    ...
}
```

That function tells the validator what name to use in error messages.

Without this function, the validator might say:

```txt
Email must be valid
```

But your API uses JSON field names, so you want:

```txt
email must be valid
```

That is why the test exists:

```go
func TestValidateStructUsesJSONFieldNames(t *testing.T) {
```

The test expects:

```txt
email must be valid
```

not:

```txt
Email must be valid
```

### What Does `field.Tag.Get("json")` Do?

A struct tag is this part:

```go
`json:"email" validate:"required,email,max=255"`
```

The tag is attached to the struct field.

This line:

```go
field.Tag.Get("json")
```

returns:

```txt
email
```

For this:

```go
Name *string `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
```

it returns:

```txt
name,omitempty
```

So the code does:

```go
name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
```

`strings.Cut("name,omitempty", ",")` splits once at the comma.

It returns three values:

```go
before, after, found
```

So:

```go
name, _, _ := strings.Cut(...)
```

means:

```txt
Give me the part before the comma.
Ignore the part after the comma.
Ignore whether a comma was found.
```

The underscores are throwaway variables.

So:

```txt
json:"name,omitempty" -> "name"
json:"email" -> "email"
```

Then:

```go
if name == "-" {
    return ""
}
return name
```

If a field has:

```go
json:"-"
```

that means "do not include this field in JSON." So this function returns an empty name.

### Why Validation Is Global

```go
var validate = newValidator()
```

This creates one validator instance for the package.

In Node terms:

```ts
const validate = createValidator()
```

The validator can be reused safely. You do not need to recreate it for every request.

### `ValidateStruct(input any)`

```go
func ValidateStruct(input any) error {
    if err := validate.Struct(input); err != nil {
        ...
    }

    return nil
}
```

`any` means the function can accept any input type.

That makes sense because this helper should validate different DTO structs:

```go
ValidateStruct(EmailSignupInput{})
ValidateStruct(EmailLoginInput{})
ValidateStruct(VerifyOTPInput{})
```

Inside:

```go
if err := validate.Struct(input); err != nil {
```

The validator uses reflection to inspect fields and tags.

If validation fails:

```go
if validationErrors, ok := err.(validator.ValidationErrors); ok && len(validationErrors) > 0 {
    return apperror.BadRequest(validationErrorMessage(validationErrors[0]))
}
```

This only returns the first validation error.

So if email and password are both bad, the API responds with the first one the validator reports.

That is a product/API choice. You could later change it to return all validation errors.

### The Message Builder

```go
func validationErrorMessage(err validator.FieldError) string {
    field := err.Field()

    switch err.Tag() {
    case "required":
        return fmt.Sprintf("%s is required", field)
    case "email":
        return fmt.Sprintf("%s must be valid", field)
    case "min":
        return fmt.Sprintf("%s must be at least %s characters", field, err.Param())
    ...
    }
}
```

`err.Field()` is the JSON field name because of the `RegisterTagNameFunc` setup.

`err.Tag()` is the validation rule that failed:

```txt
required
email
min
max
len
numeric
```

`err.Param()` is the value attached to a rule:

```txt
min=8 -> "8"
max=255 -> "255"
len=6 -> "6"
```

So:

```go
Password string `validate:"required,min=8,max=72"`
```

can produce:

```txt
password must be at least 8 characters
```

This is basically a tiny custom error-message layer over the validator library.

## `helpers/response.go`: JSON IO

File:

```txt
apps/api/internal/helpers/response.go
```

This file contains two big helpers:

```go
ReadJSON(...)
WriteJSON(...)
```

### `WriteJSON`

```go
func WriteJSON(w http.ResponseWriter, status int, data Envelope, headers http.Header) error {
    js, err := json.MarshalIndent(data, "", "\t")
    ...
    maps.Copy(w.Header(), headers)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _, err = w.Write(js)
    return err
}
```

Hono equivalent:

```ts
return c.json(data, status, headers)
```

What it does:

1. Convert Go value to JSON.
2. Add a newline.
3. Copy custom headers.
4. Set `Content-Type`.
5. Set status.
6. Write body.

`json.MarshalIndent` pretty-prints JSON. For production APIs you might use `json.Marshal` instead to avoid extra whitespace, but this is fine for learning and debugging.

### `maps.Copy`

```go
maps.Copy(w.Header(), headers)
```

Copies headers from one map into another.

`w.Header()` returns the response headers map.

One caution: if `headers` is `nil`, `maps.Copy` with a nil source is fine. It copies nothing.

### `ReadJSON`

```go
func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
```

This reads JSON into `dst`.

Important:

```go
r.Body = http.MaxBytesReader(w, r.Body, MAX_JSON_BODY_SIZE)
```

Limits body size to 1 MB.

```go
dec := json.NewDecoder(r.Body)
dec.DisallowUnknownFields()
```

Creates a decoder and rejects unknown fields.

So if signup expects:

```json
{
  "email": "...",
  "password": "..."
}
```

and the client sends:

```json
{
  "email": "...",
  "password": "...",
  "admin": true
}
```

the API rejects it.

That is good strict API behavior.

Then:

```go
err := dec.Decode(dst)
```

fills the struct you passed.

Because `dst` is a pointer:

```go
&input
```

the decoder can mutate it.

### The Error Cases

The switch catches common JSON parsing errors and turns them into friendly messages:

- bad JSON syntax
- unexpected EOF
- wrong JSON type for a field
- empty body
- unknown field
- body too large
- invalid destination passed by the programmer

This:

```go
case errors.As(err, &invalidUnmarshalError):
    panic(err)
```

means:

```txt
If a programmer called ReadJSON incorrectly, crash loudly.
```

For example:

```go
helpers.ReadJSON(w, r, input)
```

instead of:

```go
helpers.ReadJSON(w, r, &input)
```

That is not a client error. That is a developer bug.

### Rejecting Multiple JSON Values

After decoding once:

```go
err = dec.Decode(&struct{}{})
if !errors.Is(err, io.EOF) {
    return errors.New("body must only contain a single JSON value")
}
```

This rejects bodies like:

```json
{"email":"a@example.com"}{"email":"b@example.com"}
```

Many simple JSON parsers accidentally accept the first value and ignore the rest. This helper is stricter.

## `auth/service.go`: Business Logic

File:

```txt
apps/api/internal/auth/service.go
```

This is where the signup rules live.

```go
func (s *AuthService) SignupWithEmail(ctx context.Context, input EmailSignupInput) (*user.User, error) {
```

Again, `(s *AuthService)` is the receiver. In TypeScript:

```ts
class AuthService {
  async signupWithEmail(input) {}
}
```

### The Service Struct

```go
type AuthService struct {
    userRepo UserRepository
    otpRepo  OTPRepository
    logger   *slog.Logger
}
```

This service has access to:

- user repository
- OTP repository
- logger

It does not know about HTTP. That is important.

No `http.ResponseWriter`.
No `*http.Request`.
No JSON response writing.

That keeps it reusable.

In Node terms, this should not know about Hono's `Context`.

Good:

```ts
authService.signupWithEmail(input)
```

Less good:

```ts
authService.signupWithEmail(c)
```

The current Go code follows the good shape.

### Signup Logic

```go
existingUser, err := s.userRepo.FindByEmail(ctx, input.Email)
if err != nil {
    return nil, err
}
if existingUser != nil {
    return nil, apperror.Conflict("user already exists.")
}
```

This checks for an existing user.

Because `FindByEmail` returns `nil, nil` when no user exists, the service can do:

```go
if existingUser != nil
```

In TypeScript:

```ts
const existingUser = await userRepo.findByEmail(input.email)
if (existingUser) {
  throw new ConflictError('user already exists.')
}
```

Then:

```go
id := helpers.NewUserID()
```

Generates an ID like:

```txt
user_<cuid2>
```

Then:

```go
hp, err := helpers.NewHasher().Hash(input.Password)
```

Hashes the password.

The variable name `hp` means hashed password. You may want to rename it to `passwordHash` later for readability.

Then:

```go
dto := user.CreateUserParams{
    ID:           id,
    Name:         input.Name,
    Email:        input.Email,
    PasswordHash: hp,
}
```

This creates a repository input object.

The service passes it to:

```go
user, err := s.userRepo.Create(ctx, dto)
```

That creates the database row.

### Why The Service Returns `*user.User`

The service returns the internal user model:

```go
(*user.User, error)
```

Then the handler maps it to a response DTO.

That is okay for this early codebase. Later, if service outputs become more complex, you may decide services should return their own result DTOs rather than database models.

For now:

```txt
repository returns DB model
service returns DB model
handler maps DB model to API response
```

is understandable and workable.

## `helpers/hasher.go`: Password Hashing

File:

```txt
apps/api/internal/helpers/hasher.go
```

This uses Argon2id:

```go
hash := argon2.IDKey(...)
```

Argon2id is a password hashing algorithm. The point is to make password hashes slow and expensive enough that stolen hashes are harder to brute force.

The constants:

```go
memory      uint32 = 64 * 1024 //64mb
iterations  uint32 = 3
parallelism uint8  = 2
saltLength  uint32 = 16
keyLength   uint32 = 32
```

define hashing parameters:

- `memory`: memory cost, here 64 MB.
- `iterations`: number of passes.
- `parallelism`: lanes/threads.
- `saltLength`: random salt length.
- `keyLength`: output hash length.

### Hashing

```go
func (h *Hasher) Hash(plain string) (string, error) {
    salt, err := randomBytes(saltLength)
    ...
    hash := argon2.IDKey(...)
    encodedHash := base64.RawStdEncoding.EncodeToString(hash)
    encodedSalt := base64.RawStdEncoding.EncodeToString(salt)

    return encodedSalt + "." + encodedHash, nil
}
```

This returns:

```txt
base64salt.base64hash
```

Example shape:

```txt
mDgqD3r...JXo.pPFfYk...z9M
```

The salt is not secret. It must be stored so you can verify the password later.

### Comparing

```go
func (h *Hasher) Compare(encodedHash string, plain string) error {
    salt, hash, err := decodeHash(encodedHash)
    ...
    otherHash := argon2.IDKey(...)

    if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
        return nil
    }
    return ErrInvalidPassword
}
```

This re-hashes the incoming plain password with the original salt and compares the result.

`subtle.ConstantTimeCompare` is used so the comparison does not leak timing information.

In Node you might use:

```ts
await argon2.verify(storedHash, password)
```

Here the project is manually storing and verifying `salt.hash`.

One note: a common production format stores algorithm parameters with the hash too, so future parameter changes are easier. This code currently uses fixed app-level constants.

## `user/repository.go`: SQL Layer

File:

```txt
apps/api/internal/user/repository.go
```

Repositories own database queries.

The user repository has:

```go
type Repository struct {
    db *sqlx.DB
}
```

It wraps the database connection pool.

### `sqlx`

This app uses:

```go
github.com/jmoiron/sqlx
```

`sqlx` is a helper library over Go's standard `database/sql`.

It gives nicer methods like:

```go
r.db.GetContext(...)
r.db.QueryRowxContext(...).StructScan(...)
```

The `Context` part means the query receives request cancellation/timeouts.

### Create User

```go
func (r *Repository) Create(ctx context.Context, params CreateUserParams) (*User, error) {
    query := `
    INSERT INTO users (id, name, email, password_hash)
    VALUES ($1, $2, $3, $4)
    RETURNING id, name, email, avatar_url, email_verified_at, created_at
    `
    var user User

    args := []any{params.ID, params.Name, params.Email, params.PasswordHash}
    err := r.db.QueryRowxContext(ctx, query, args...).StructScan(&user)
    ...
}
```

Important pieces:

- `$1`, `$2`, `$3`, `$4` are Postgres placeholders.
- `args := []any{...}` builds the values for those placeholders.
- `args...` spreads the slice into function arguments.
- `RETURNING ...` asks Postgres to return the inserted row.
- `StructScan(&user)` maps returned columns into struct fields.

`StructScan` uses the `db` tags on the model:

```go
type User struct {
    ID        string         `db:"id"`
    Name      sql.NullString `db:"name"`
    Email     string         `db:"email"`
    CreatedAt time.Time      `db:"created_at"`
}
```

So column `created_at` maps to field `CreatedAt`.

In Node with Drizzle/Kysely/Prisma you would not think about scanning as much because the library returns objects. In Go SQL code, scanning is explicit.

### Find By Email

```go
func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
    query := `
    SELECT id, name, email, avatar_url, email_verified_at, created_at
    FROM users
    WHERE email = $1
    `

    var user User
    err := r.db.GetContext(ctx, &user, query, email)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    return &user, nil
}
```

This is an important pattern:

```go
if err == sql.ErrNoRows {
    return nil, nil
}
```

The repository translates "no row" from a database error into "no user found."

In TypeScript:

```ts
const row = await db.query(...)
return row ?? null
```

### Update User

```go
UPDATE users
SET name = COALESCE($1, name),
    avatar_url = COALESCE($2, avatar_url),
    email = COALESCE($3, email),
    updated_at = NOW()
WHERE id = $4
RETURNING ...
```

`COALESCE($1, name)` means:

```txt
If $1 is not null, use $1.
Otherwise keep the existing name.
```

This is why `UpdateUserParams` uses pointers:

```go
type UpdateUserParams struct {
    ID        string
    Name      *string
    AvatarURL *string
    Email     *string
}
```

`nil` means "do not update this field."

One subtle limitation: with this pattern, you cannot set a nullable column back to SQL `NULL`, because `nil` means "keep the old value." That is fine if you want partial update behavior, but it is worth knowing.

## `user/model.go`: Database Shape

File:

```txt
apps/api/internal/user/model.go
```

```go
type User struct {
    ID              string         `db:"id"`
    Name            sql.NullString `db:"name"`
    Email           string         `db:"email"`
    PasswordHash    sql.NullString `db:"password_hash"`
    AvatarURL       sql.NullString `db:"avatar_url"`
    EmailVerifiedAt sql.NullTime   `db:"email_verified_at"`
    CreatedAt       time.Time      `db:"created_at"`
}
```

This is a database model.

It is shaped for scanning SQL rows, not for JSON responses.

That is why it has:

```go
sql.NullString
sql.NullTime
```

instead of:

```go
*string
*time.Time
```

The `db:"..."` tags are for `sqlx`, not JSON.

## `user/dto.go`: User API Shapes

File:

```txt
apps/api/internal/user/dto.go
```

```go
type CreateUserParams struct {
    ID           string
    Name         *string
    Email        string
    PasswordHash string
}
```

This is not an HTTP DTO. It is a repository parameter DTO.

It means:

```txt
These are the fields needed to create a user row.
```

Then:

```go
type UserResponse struct {
    ID              string     `json:"id"`
    Name            *string    `json:"name,omitempty"`
    Email           string     `json:"email"`
    AvatarURL       *string    `json:"avatar_url,omitempty"`
    EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
    CreatedAt       time.Time  `json:"created_at"`
}
```

This is an HTTP response DTO.

There is also:

```go
func toUserResponse(u User) UserResponse {
```

That mapper is currently unexported because it starts with lowercase `t`.

Go visibility rule:

- `toUserResponse` is private to the `user` package.
- `ToUserResponse` would be public/exported.

That matters because `auth/handler.go` cannot call `user.toUserResponse`. It has its own mapper:

```go
toAuthUserResponse(...)
```

You may eventually choose to export a shared mapper if response shapes stay identical. For now duplication is acceptable because auth responses and user responses may diverge.

## `apperror/errors.go`: App-Level Errors

File:

```txt
apps/api/internal/apperror/errors.go
```

This defines a custom error type:

```go
type Error struct {
    Code    string
    Message string
    Status  int
    Err     error
}
```

In TypeScript:

```ts
class AppError extends Error {
  code: string
  status: number
  cause?: Error
}
```

This method:

```go
func (e *Error) Error() string {
    return e.Message
}
```

makes `*Error` satisfy Go's built-in `error` interface.

In Go, an error is anything with:

```go
Error() string
```

This:

```go
func (e *Error) Unwrap() error {
    return e.Err
}
```

allows wrapped errors to work with `errors.Is` and `errors.As`.

Then helpers:

```go
func BadRequest(message string) *Error
func Unauthorized(message string) *Error
func NotFound(message string) *Error
func Conflict(message string) *Error
func Internal() *Error
```

are convenience constructors.

In Node:

```ts
throw AppError.conflict('user already exists')
```

In Go:

```go
return nil, apperror.Conflict("user already exists.")
```

Go generally returns errors instead of throwing exceptions.

## `config/config.go`: Environment Loading

File:

```txt
apps/api/internal/config/config.go
```

This reads environment variables:

```go
_ = godotenv.Load()
```

That loads `.env` if present. The underscore means "ignore the returned error."

That is intentional because `.env` may not exist in production.

Then:

```go
port, err := getInt("API_PORT", 4001)
```

reads an int environment variable with fallback.

The final config:

```go
cfg := Config{
    AppEnv: getString("APP_ENV", "development"),
    Port:   port,
    DbUrl:  getDatabaseURL(),
}
```

Then:

```go
if err := cfg.Validate(); err != nil {
    return Config{}, err
}
```

validates the config early at startup.

That is good. You want the app to fail immediately if config is bad, not fail halfway through a request.

One tiny wording mismatch:

```go
return fmt.Errorf("PORT must be between 1 and 65535")
```

The env var read is `API_PORT`, not `PORT`. You might want that error to say `API_PORT`.

## `db/db.go`: Database Connection

File:

```txt
apps/api/internal/db/db.go
```

```go
func Open(databaseUrl string) (*sqlx.DB, error) {
    db, err := sqlx.Open("postgres", databaseUrl)
    ...
}
```

This creates a Postgres connection pool.

Important import:

```go
_ "github.com/lib/pq"
```

The underscore import means:

```txt
Import this package only for its side effects.
```

The `lib/pq` package registers the Postgres driver with Go's database system. You do not call it directly, but `sqlx.Open("postgres", ...)` needs it to be registered.

Then:

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(25)
db.SetConnMaxIdleTime(15 * time.Minute)
```

configure the pool.

Then:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
if err := db.PingContext(ctx); err != nil {
    db.Close()
    return nil, err
}
```

This verifies the database is reachable during startup.

In Node:

```ts
await db.ping({ timeout: 5000 })
```

If ping fails, app startup fails.

## `platform/logger/logger.go`: Logger Setup

File:

```txt
apps/api/platform/logger/logger.go
```

```go
func New(env string) *slog.Logger {
    var handler slog.Handler

    if env == "development" {
        handler = slog.NewTextHandler(os.Stdout, ...)
    } else {
        handler = slog.NewJSONHandler(os.Stdout, ...)
    }

    return slog.New(handler)
}
```

In development logs are human-readable text.

In test/production they are JSON.

`slog` is Go's standard structured logging package.

This:

```go
appLogger.Error("failed to connect to database", "error", err)
```

is structured logging. The message is:

```txt
failed to connect to database
```

and `"error", err` is a key/value pair.

## `helpers/ids.go`: IDs

File:

```txt
apps/api/internal/helpers/ids.go
```

```go
func NewUserID() string {
    id := cuid2.Generate()
    return "user_" + id
}
```

This creates prefixed IDs.

Example:

```txt
user_ckxyz...
session_ckxyz...
```

This is similar to using IDs like:

```txt
usr_...
sess_...
```

Prefixing IDs makes logs and debugging easier because you can tell what kind of object an ID refers to.

## `helpers/otp.go`: OTP Generation

File:

```txt
apps/api/internal/helpers/otp.go
```

```go
func GenerateOTP() (string, error) {
    gen := NewGenerator(6)
    return gen.Generate()
}
```

This creates a 6-digit code.

The important bit:

```go
n, err := rand.Int(rand.Reader, max)
```

This uses cryptographic randomness, not `math/rand`.

Then:

```go
format := fmt.Sprintf("%%0%dd", g.length)
return fmt.Sprintf(format, n), nil
```

This pads with leading zeroes.

So the generated code can be:

```txt
004213
```

not:

```txt
4213
```

## `auth/repository.go`: OTP Repository

File:

```txt
apps/api/internal/auth/repository.go
```

Current code:

```go
func (r *Repository) CreateOTP(ctx context.Context, email string, otpHash string) error {
    query := `
    INSERT INTO otps (email, otp_hash)
    VALUES ($1, $2)
    `
    _, err := r.db.ExecContext(ctx, query, email, otpHash)
    return err
}
```

This is intended to insert OTPs.

But right now it does not match the migration.

The migration defines:

```sql
CREATE TABLE otps (
  id bigserial primary key,
  user_id text not null references users(id) on delete cascade,
  email text not null,
  otp_hash bytea not null,
  purpose text not null,
  attempts int not null default 0,
  consumed_at timestamptz,
  expires_at timestamptz not null,
  created_at timestamptz not null default now(),
  constraint otps_purpose_check check (purpose in ('signup', 'password_reset')),
  ...
);
```

Required columns with no default:

- `user_id`
- `email`
- `otp_hash`
- `purpose`
- `expires_at`

The repository only inserts:

- `email`
- `otp_hash`

So `CreateOTP` will fail if called against this schema.

Also:

```sql
otp_hash bytea not null
```

but Go passes:

```go
otpHash string
```

Postgres may or may not cast that how you expect. This should probably become `[]byte` or the DB column should become `text`, depending on how you want to store OTP hashes.

There is another mismatch:

```go
const (
    OTPPurposeEmailVerification OTPPurpose = "email_verification"
    OTPPurposePasswordReset     OTPPurpose = "password_reset"
)
```

but the migration check says:

```sql
purpose in ('signup', 'password_reset')
```

So Go says `email_verification`, SQL allows `signup`.

That needs to be reconciled before OTP verification is implemented.

This is exactly the kind of thing a walkthrough should point out because it will bite you later.

## `auth/model.go`: OTP Model

File:

```txt
apps/api/internal/auth/model.go
```

```go
type OTP struct {
    ID         string       `db:"id"`
    UserID     string       `db:"user_id"`
    Email      string       `db:"email"`
    OtpHash    string       `db:"otp_hash"`
    Purpose    OTPPurpose   `db:"purpose"`
    ExpiresAt  time.Time    `db:"expires_at"`
    CreatedAt  time.Time    `db:"created_at"`
    ConsumedAt sql.NullTime `db:"consumed_at"`
    Attempts   int          `db:"attempts"`
}
```

This is the database shape for OTPs.

There are a few likely cleanup items:

- Migration says `id bigserial`, but model says `ID string`. That should probably be `int64`.
- Migration says `otp_hash bytea`, but model says `OtpHash string`.
- Go constant says `email_verification`, migration allows `signup`.

This area is not finished yet.

## Migration: Database Schema

File:

```txt
apps/api/migrations/20260526104632_create_users_and_sessions.sql
```

The migration creates:

- `users`
- `sessions`
- `otps`
- `oauth_accounts`

### Users

```sql
CREATE TABLE users (
  id text primary key,
  name text,
  email text not null,
  password_hash text,
  avatar_url text,
  email_verified_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
```

Notice:

```sql
password_hash text
```

is nullable. That likely supports OAuth users who do not have a password.

Then:

```sql
create unique index users_lower_email_unique_idx on users(lower(email));
```

This makes email uniqueness case-insensitive.

So:

```txt
Temi@example.com
temi@example.com
```

cannot both exist.

The handler also lowercases email before storing it, which is good.

### Sessions

```sql
CREATE TABLE sessions (
  id text primary key,
  user_id text not null references users(id) on delete cascade,
  user_agent text,
  token_hash bytea not null,
  ip_address inet,
  expires_at timestamptz not null,
  revoked_at timestamptz,
  created_at timestamptz not null default now(),
  constraint expires_at_greater_than_created_at check (expires_at>created_at)
);
```

This is not wired into Go yet, but the schema is ready for session auth.

The important idea:

- Store token hash, not raw token.
- Sessions belong to users.
- Deleting a user deletes sessions.
- Sessions can expire or be revoked.

### OAuth Accounts

```sql
CREATE TABLE oauth_accounts (
  provider oauth_providers not null,
  provider_user_id text not null,
  user_id text not null references users(id) on delete cascade,
  created_at timestamptz not null default now(),
  primary key (provider, provider_user_id)
);
```

This lets one user be linked to external providers like Google or GitHub.

## A Full Signup Request, Top To Bottom

Imagine the frontend sends:

```http
POST /auth/signup/email
Content-Type: application/json
```

```json
{
  "name": " Temi ",
  "email": " TEMI@example.com ",
  "password": "supersecret123"
}
```

### 1. Router Matches The Route

`server.go` registered:

```go
opts.Mux.HandleFunc("POST /auth/signup/email", opts.AuthHandler.SignupWithEmail)
```

The mux calls:

```go
authHandler.SignupWithEmail(w, r)
```

### 2. Handler Reads JSON

```go
var input EmailSignupInput
helpers.ReadJSON(w, r, &input)
```

Now:

```go
input.Name     -> pointer to " Temi "
input.Email    -> " TEMI@example.com "
input.Password -> "supersecret123"
```

### 3. Handler Normalizes

```go
input = normalizeEmailSignupInput(input)
```

Now:

```go
input.Name     -> pointer to "Temi"
input.Email    -> "temi@example.com"
input.Password -> "supersecret123"
```

### 4. Handler Validates

```go
helpers.ValidateStruct(input)
```

The validator checks:

```go
Name     omitempty,min=2,max=255
Email    required,email,max=255
Password required,min=8,max=72
```

If invalid, handler returns:

```json
{
  "error": {
    "code": "BAD_REQUEST",
    "message": "email must be valid"
  }
}
```

### 5. Handler Calls Service

```go
createdUser, err := h.service.SignupWithEmail(r.Context(), input)
```

### 6. Service Checks Duplicate User

```go
existingUser, err := s.userRepo.FindByEmail(ctx, input.Email)
```

Repository runs:

```sql
SELECT id, name, email, avatar_url, email_verified_at, created_at
FROM users
WHERE email = $1
```

If found:

```go
return nil, apperror.Conflict("user already exists.")
```

Handler turns that into HTTP 409.

### 7. Service Hashes Password

```go
hp, err := helpers.NewHasher().Hash(input.Password)
```

The raw password never gets stored.

### 8. Service Creates User

```go
dto := user.CreateUserParams{
    ID:           helpers.NewUserID(),
    Name:         input.Name,
    Email:        input.Email,
    PasswordHash: hp,
}
user, err := s.userRepo.Create(ctx, dto)
```

Repository runs:

```sql
INSERT INTO users (id, name, email, password_hash)
VALUES ($1, $2, $3, $4)
RETURNING id, name, email, avatar_url, email_verified_at, created_at
```

### 9. SQL Result Becomes `user.User`

`StructScan(&user)` fills:

```go
type User struct {
    ID              string
    Name            sql.NullString
    Email           string
    AvatarURL       sql.NullString
    EmailVerifiedAt sql.NullTime
    CreatedAt       time.Time
}
```

### 10. Handler Maps To JSON DTO

```go
toAuthUserResponse(createdUser)
```

The result becomes:

```json
{
  "data": {
    "user": {
      "id": "user_...",
      "name": "Temi",
      "email": "temi@example.com",
      "created_at": "2026-06-03T..."
    }
  }
}
```

## Important Go Concepts In This Codebase

### Pointers

```go
*string
```

means "pointer to a string."

In this codebase, pointers usually mean optional values:

```go
Name *string
```

can be:

```go
nil
```

or:

```go
&someString
```

TypeScript mental model:

```ts
string | undefined
```

### `nil`

`nil` is Go's absence value for pointers, interfaces, maps, slices, channels, and function values.

It is kind of like `null`, but only certain types can be nil.

A plain string cannot be nil:

```go
var email string // empty string, not nil
```

A string pointer can be nil:

```go
var name *string // nil
```

### Struct Tags

```go
Email string `json:"email" validate:"required,email"`
```

Struct tags are runtime metadata.

Libraries inspect them using reflection.

Here:

- `json` tag is used by `encoding/json`.
- `validate` tag is used by `go-playground/validator`.
- `db` tag is used by `sqlx`.

### Receivers

```go
func (h *Handler) SignupWithEmail(...)
```

The `(h *Handler)` part is like `this` in a class method.

### Interfaces

```go
type UserRepository interface {
    FindByEmail(...) (*user.User, error)
}
```

Interfaces describe behavior, not inheritance.

If a type has the required methods, it satisfies the interface automatically.

No `implements` keyword needed.

### Error Returns

Go does this:

```go
result, err := doThing()
if err != nil {
    return nil, err
}
```

Instead of:

```ts
try {
  const result = await doThing()
} catch (err) {
  ...
}
```

Errors are ordinary values.

### `context.Context`

Context carries cancellation, deadlines, and request-scoped values.

This app mainly uses it for request cancellation:

```go
r.Context()
```

Then passes it down:

```go
service -> repository -> database query
```

That is good practice.

### `defer`

In `main.go`:

```go
defer database.Close()
```

`defer` runs when the current function exits.

Node equivalent is roughly:

```ts
try {
  ...
} finally {
  database.close()
}
```

### Blank Identifier `_`

```go
name, _, _ := strings.Cut(...)
```

means "ignore these returned values."

Another example:

```go
_ = godotenv.Load()
```

means "call this, but ignore the result."

### Exported vs Private Names

In Go, capitalization controls visibility.

Public/exported:

```go
NewHandler
AuthService
EmailSignupInput
```

Private to package:

```go
normalizeEmailSignupInput
toAuthUserResponse
validate
```

There is no `export` keyword.

## How To Add A New Route

Suppose you want:

```txt
POST /auth/login/email
```

The current architecture wants you to do this:

### 1. Add/confirm input DTO

In `auth/dto.go`:

```go
type EmailLoginInput struct {
    Email    string `json:"email" validate:"required,email,max=255"`
    Password string `json:"password" validate:"required"`
}
```

Already exists.

### 2. Add service method

In `auth/service.go`:

```go
func (s *AuthService) LoginWithEmail(ctx context.Context, input EmailLoginInput) (*user.User, error) {
    // find user
    // get password hash
    // compare password
    // create session eventually
    // return user/session
}
```

### 3. Add handler method

In `auth/handler.go`:

```go
func (h *Handler) LoginWithEmail(w http.ResponseWriter, r *http.Request) {
    var input EmailLoginInput
    if err := helpers.ReadJSON(w, r, &input); err != nil {
        h.writeError(w, apperror.BadRequest(err.Error()))
        return
    }

    input.Email = strings.ToLower(strings.TrimSpace(input.Email))

    if err := helpers.ValidateStruct(input); err != nil {
        h.writeError(w, err)
        return
    }

    result, err := h.service.LoginWithEmail(r.Context(), input)
    if err != nil {
        h.writeError(w, err)
        return
    }

    helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{
        "data": ...
    }, nil)
}
```

### 4. Register route

In `server.go`:

```go
opts.Mux.HandleFunc("POST /auth/login/email", opts.AuthHandler.LoginWithEmail)
```

That is the pattern.

For every feature:

```txt
DTO -> handler -> service -> repository -> model/SQL -> response mapper
```

## Things I Would Keep An Eye On

These are not blockers for understanding, but they are worth knowing before building more.

### OTP Code Is Not Schema-Compatible Yet

As explained above, OTP repository/model/constants do not match the migration. Fix that before relying on OTPs.

### Signup Does Not Create Session Yet

`AuthResponse` has:

```go
// Session AuthSessionResponse `json:"session,omitempty"`
```

So the intended design probably includes returning a session after signup/login, but it is not implemented.

### Database Errors Are Not Yet Translated

`user.Repository.Create` returns raw DB errors.

Because there is a unique index on lower email, duplicate email could also be caught by the DB. Right now the service checks first, but race conditions are still possible:

```txt
Request A checks email, no user
Request B checks email, no user
Request A inserts
Request B inserts, DB unique index fails
```

The DB protects you, but the app may return a raw internal error unless you translate the unique constraint error into `apperror.Conflict`.

### `AuthService` Creates A Hasher Inline

```go
helpers.NewHasher().Hash(input.Password)
```

This is okay for now.

If you want easier tests, you could inject a hasher interface into `AuthService`, similar to repositories.

### Service Naming

File is:

```txt
apps/api/internal/user/services.go
```

but auth uses:

```txt
service.go
```

Consistency helps. If `user/services.go` is empty or future work, consider making naming consistent.

### Response Mapper Duplication

`auth/handler.go` has `toAuthUserResponse`.

`user/dto.go` has `toUserResponse`.

They do basically the same nullable-field conversion. That is fine now. Later, if the shapes stay identical, you can share it.

## The Mental Model

Think of this Go backend as a Hono app where the framework conveniences have been unfolded into explicit pieces.

```txt
Hono app/router        -> http.ServeMux
Hono context `c`       -> http.ResponseWriter + *http.Request
Route handler          -> method on Handler struct
Middleware-ish helpers -> helpers.ReadJSON, helpers.WriteJSON, ValidateStruct
Zod schema             -> Go struct tags + validator library
Service class          -> Go struct with methods
Repository class       -> Go struct with db field and SQL methods
Dependency injection   -> manually passing things in main.go
Throw AppError         -> return apperror.Error
Request abort signal   -> context.Context
Database row object    -> struct scanned by sqlx
null/undefined         -> nil pointers or sql.NullString/sql.NullTime
```

The thing that feels weird at first is that Go makes the boundaries very visible. You have to say:

```txt
This is the HTTP shape.
This is the DB shape.
This is the service dependency.
This is the repository dependency.
This is the app startup graph.
```

That can feel like boilerplate, especially coming from Node. But the upside is that once you understand the wiring, there is very little hidden magic. You can follow the request path with your finger from route to SQL and back.

For this project, the next code you write should usually follow this rule:

```txt
If it talks HTTP, put it in a handler.
If it decides business rules, put it in a service.
If it talks SQL, put it in a repository.
If it is a JSON shape, put it in a DTO.
If it is a database row shape, put it in a model.
If it wires objects together, put it in main.go or server setup.
```

That is the backbone of this API.
