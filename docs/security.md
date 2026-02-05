# Watchflare — Security Model

---

## Layers at a Glance

| Layer | Mechanism | Applies to |
|-------|-----------|------------|
| Transport | TLS 1.3 (mandatory, no fallback) | All agent ↔ backend communication |
| Agent registration | One-time token, SHA-256 hashed in DB | `RegisterServer` RPC |
| Agent authentication | HMAC-SHA256 per request | All RPCs after registration |
| Anti-replay | Timestamp window (default ±5 min) | All HMAC-signed RPCs |
| Web user authentication | JWT in HttpOnly cookie | All protected HTTP endpoints |
| Password storage | bcrypt (cost 10) | User passwords |
| Certificate authority | Self-signed PKI or operator-supplied | gRPC server identity |

---

## TLS & Certificate Authority

The backend's gRPC server requires TLS 1.3. Both `MinVersion` and `MaxVersion` in
the Go `tls.Config` are set to `tls.VersionTLS13` — no negotiation to an older
protocol is possible.

### Auto mode (default)

On first startup the backend generates:
- A **CA certificate** — self-signed, RSA-4096, valid 10 years.
- A **server certificate** — signed by the CA, RSA-4096, valid 5 years.
  SANs include `watchflare` and `localhost`.

Both are persisted to `./pki/` (configurable via `TLS_PKI_DIR`). Private keys are
written with mode `0600`.

### Custom mode

The operator provides three files: `ca.pem`, `server.pem`, `server-key.pem`. The
PKI module validates they exist and can be loaded before the server starts.

### Agent-side trust

During registration the backend sends its CA certificate (PEM) to the agent.
The agent saves it to `/etc/watchflare/ca.pem`. All subsequent TLS connections
verify the backend's certificate against this CA. This single file is the agent's
trust anchor.

---

## Agent Registration

Registration solves a bootstrap problem: how does the agent prove it is authorised
to connect, when it has no shared secret yet?

### Token lifecycle

```
1. Admin creates a server in the dashboard.

2. Backend generates:
     token  = "wf_reg_" + hex(16 random bytes)   →  38 chars
     key    = hex(32 random bytes)                →  64 chars (AES-256)

3. Only SHA-256(token) is stored in the database.
   The plaintext token is returned to the admin once.

4. The admin passes the token to the agent (via the install command).

5. The agent sends the plaintext token in RegisterServer.

6. Backend: SHA-256(received token) → DB lookup.
   Checks:
     - Token exists
     - Not expired (24 h window)
     - Server status is "pending" or "expired"
     - IP matches configured_ip (unless allow_any_ip is set)

7. On success:
     - Token and expiry fields are set to NULL in the DB.
     - Server status → "online".
     - Backend returns agent_id, agent_key, and the CA cert.
```

The token is never stored in plaintext anywhere. Even if the database is compromised,
tokens cannot be extracted from it.

### The InsecureSkipVerify window

The agent's first connection uses `InsecureSkipVerify: true` because it does not yet
have the CA certificate. This is a deliberate, single-RPC window. After registration
the agent pins the CA cert and all connections are fully verified.

Mitigation: the token itself is the authentication credential for this call. An
attacker who can MITM this connection would also need to know or intercept the token
to impersonate the agent. In most deployment scenarios the token is delivered
out-of-band (pasted into a terminal on the server itself).

---

## Post-Registration: HMAC-SHA256

Every gRPC call after registration is signed. The backend's interceptor rejects any
unsigned request (except `RegisterServer`).

### Signature construction

```
payload = [timestamp: 8 bytes, big-endian int64]
        + "|"
        + agent_id
        + "|"
        + proto.Marshal(request_message)

signature = HMAC-SHA256(agent_key, payload)
```

The agent attaches two gRPC metadata headers:
- `x-watchflare-hmac`  — hex(signature)
- `x-watchflare-timestamp` — decimal string of the same Unix timestamp

### Why 8-byte binary timestamp?

A decimal string like `"1234567890"` is susceptible to whitespace, leading-zero, or
encoding tricks that could cause the agent and backend to disagree on the signed
payload while producing the same HMAC. The fixed 8-byte big-endian representation
is unambiguous.

### Backend validation

1. Extract `x-watchflare-hmac` and `x-watchflare-timestamp` from metadata.
2. Use reflection to extract `agent_id` and `timestamp` from the protobuf message.
   (This keeps the interceptor generic — it does not need to know the concrete type
   of each RPC's request.)
3. Look up `agent_key` from the `servers` table by `agent_id`.
4. Validate the timestamp (next section).
5. Recompute the HMAC with the same payload construction.
6. Compare using `hmac.Equal` — constant-time, immune to timing side-channels.

---

## Anti-Replay Protection

The timestamp embedded in every signed request prevents replay attacks.

The backend checks:
```
now - GRPCTimestampWindow  ≤  request.timestamp  ≤  now + GRPCTimestampWindow
```

Default window: **300 seconds (5 minutes)**. Configurable via `GRPC_TIMESTAMP_WINDOW`.

A captured request becomes invalid after the window passes. An attacker cannot replay
it without knowing `agent_key` to re-sign with a fresh timestamp.

**Limitation:** Within the window, a captured request is technically replayable.
Eliminating this entirely would require server-side nonce tracking (state per agent).
The current approach matches standard practice for this threat model.

---

## Web User Authentication

The web dashboard authenticates via JWT stored in a browser cookie.

### Token generation

- Algorithm: HS256 (HMAC-SHA256)
- Claims: `user_id` (string), `exp` (7 days from issuance)
- Signing key: `JWT_SECRET` from environment variables

`JWT_SECRET` requirements:
- Must be present — startup is fatal otherwise.
- Must be ≥ 32 characters.
- Startup warns if it contains any of: `secret`, `password`, `admin`, `test`, `dev`,
  `change`, `please`.

### Cookie configuration

| Attribute | Value | Effect |
|-----------|-------|--------|
| `HttpOnly` | true | JavaScript cannot read the cookie — XSS cannot steal the token |
| `SameSite` | Lax | Cookie is sent on same-site requests and top-level navigations; not on cross-site sub-requests |
| `Secure` | true (production only) | Cookie only sent over HTTPS |
| `Path` | `/` | Available on all routes |
| `MaxAge` | 604800 (7 days) | Browser discards the cookie after one week |

### Per-request validation

The `AuthMiddleware` runs on every protected endpoint:
1. Read the `jwt_token` cookie.
2. Parse the JWT, verify the signature against `JWT_SECRET`.
3. Check the `exp` claim — reject if expired.
4. Extract `user_id`.
5. **Query the database** to confirm the user still exists.

Step 5 is critical: if an admin deletes a user account, any existing JWT for that
user is immediately invalid — even if the token itself has not expired.

### Registration is single-user

`POST /auth/register` checks whether any users exist before creating one. If a user
already exists, registration is rejected. The `GET /auth/setup-required` endpoint
(public, no auth) returns `{"setup_required": true/false}` so the frontend knows
whether to show the registration page or the login page.

---

## Key Management Summary

| Secret | How generated | Where stored | Lifetime | Access requirement |
|--------|---------------|--------------|----------|--------------------|
| Registration token | `crypto/rand`, 16 bytes | DB: SHA-256 hash only | 24 h, cleared on use | Shown once to operator |
| Agent key | `crypto/rand`, 32 bytes | DB: hex string | Until server is deleted | DB access |
| JWT secret | Operator-provided | Environment variable | Operator-managed | Server process env |
| CA private key | RSA-4096 | `./pki/ca-key.pem` (0600) | 10 years | Filesystem |
| Server private key | RSA-4096 | `./pki/server-key.pem` (0600) | 5 years | Filesystem |
| Agent config (incl. key) | Saved during registration | `/etc/watchflare/agent.conf` (0640) | Until re-registered | Filesystem |

**The database is a critical security boundary.** It holds `agent_key` values in
plain text (necessary for HMAC verification). Access to the database grants the
ability to impersonate any agent.

---

## Data Protection at Rest

| Data | Protection |
|------|------------|
| User passwords | bcrypt hash, cost 10. Never serialised in API responses (`json:"-"`) |
| Registration tokens | SHA-256 hashed before storage. Plaintext never persisted |
| Agent keys | Stored in DB. DB access controls are the protection boundary |
| Agent config file | Mode 0640, directory mode 0750 |
| TLS private keys | Mode 0600 |
| Metrics / packages | Stored in PostgreSQL. Access controlled at the DB level |
