# Freelance Dispatcher User Flow — Codebase Report

Structured picture of what exists and what is missing for the freelance dispatcher flow.

---

## 1. Auth & registration

### How user registration works
- **Drivers:** `internal/server/handlers/auth.go` (SendOTP, VerifyOTP) + `internal/server/handlers/registration.go` (Start, GeoPush, TransportType). Flow: phone → OTP verify → if new: session_id → POST `/v1/registration/start` (session_id, name, oferta_accepted) → driver created in `drivers` table; auth is **phone + OTP** (no password at start).
- **Freelance dispatchers:** `internal/server/handlers/dispatcher_auth.go` (SendOTP, VerifyOTP, LoginPassword, ResetPassword*, Logout) + `internal/server/handlers/dispatcher_registration.go` (Complete). Flow: phone → OTP verify → if new: session_id → POST `/v1/dispatchers/registration/complete` (session_id, name, **password**, passport_series, passport_number, pinfl) → row in **`freelance_dispatchers`**. Auth: **phone + OTP** or **phone + password**.
- **Company users (app users):** `internal/server/handlers/company_user_auth.go` (SendOTP, VerifyOTP) + `internal/server/handlers/company_user_registration.go` (Complete). Flow: phone → OTP → POST `/v1/company-users/registration/complete` → row in **`company_users`**. Used for Broker/Shipper/Carrier company staff (roles in `user_company_roles` + `app_roles`).

**Registration:** email is **not** used for drivers or freelance dispatchers; **phone** is the identifier. Password is used only for **freelance dispatchers** (and company users have password_hash). There is **no** “freelance dispatcher” vs “dispatcher” type at registration: a single “dispatcher” role is issued for anyone in `freelance_dispatchers`.

### Dispatcher role/type at registration
- **Yes:** After `/v1/dispatchers/registration/complete`, JWT is issued with **role `"dispatcher"`** (see `dispatcher_registration.go` and `security/jwt.go`). There is no separate “company dispatcher” type at registration; company-side dispatchers are **company_users** with app_roles such as Dispatcher / TopDispatcher.

### Login and tokens
- **Drivers:** VerifyOTP or (after registration) OTP again → JWT via `jwtm.Issue("driver", id)`. Refresh: POST `/v1/auth/refresh` (same as below; works for any role).
- **Dispatchers:** VerifyOTP or LoginPassword → `jwtm.Issue("dispatcher", id)` in `dispatcher_auth.go`. **No dispatcher-specific refresh route**; they can use **POST `/v1/auth/refresh`** with their refresh_token (AuthHandler.Refresh uses `claims.Role` and re-issues for that role).
- **Company users:** VerifyOTP → JWT with role `"user"`; optional **company_id** in access token after “switch company” (see below).
- **Admins:** POST `/v1/admin/auth/login/password` → JWT role `"admin"`.

**Token issuance:** `internal/security/jwt.go`: `Issue(role, userID)` and `IssueWithCompany(role, userID, companyID)`. Access claims: `role`, `user_id`, and optionally **`company_id`** (only used for app users after switch-company). There is **no** “company token” or company-scoped token for **freelance dispatchers**; they are never linked to a company in the current code.

---

## 2. Companies & roles

### Where companies are created
- **Admin flow:** `internal/server/handlers/admin_companies.go` — POST `/v1/admin/companies` (create company without owner); then PATCH `/v1/admin/companies/:id/owner` with `owner_id` = `company_users.id` (user with role OWNER). Owner is stored in `companies.owner_id`; `company_users.company_id` is also set. Tables/schema: `internal/infra/companies_schema.go` + migrations (e.g. `000021_company_tz_...`) add `owner_id`, `company_type`, etc.
- **TZ / “create own Broker company”:** `internal/server/handlers/company_tz.go` — POST `/v1/companies` (RequireAppUser). Handler `CreateCompany` uses `companies.CreateByOwner` (name, type SHIPPER/CARRIER/BROKER, owner_id = current app user), then adds row to **`user_company_roles`** with **Owner** role. So “create own company” is for **company_users** (app users), not for freelance dispatchers.

### Roles and where they are defined
- **App roles (company):** Table **`app_roles`** (id, name, description). Referenced in `internal/approles/repo.go`. Reference list in `internal/server/handlers/reference.go`: OWNER, CEO, TOP_MANAGER, TOP_DISPATCHER, DISPATCHER, MANAGER (labels in Russian). RBAC: `internal/companytz/rbac.go` — **CanInvite / CanChangeRole / CanRemove** by role name: Owner (any), CEO (any except Owner), TopManager→Manager, TopDispatcher→Dispatcher.
- **Owner:** Set (1) in **`companies.owner_id`** (admin SetOwner or CreateByOwner), and (2) in **`user_company_roles`** with role_id = Owner’s app_roles.id. `company_tz.go` resolves “is current user owner?” by `comp.OwnerID != nil && *comp.OwnerID == userID` or via UCR role.

### company_users and user_company_roles
- **company_users:** Table `company_users` (id, phone, password_hash, first_name, last_name, company_id, role, created_at, updated_at). Used for **app users** (Broker/Shipper/Carrier staff). Role here is legacy/display (e.g. OWNER); actual per-company role is in **user_company_roles**.
- **user_company_roles (UCR):** Table `user_company_roles` (user_id, company_id, role_id, assigned_by, assigned_at). Links **company_users** to companies and **app_roles**. Repo: `internal/companytz/ucr_repo.go` (Add, GetRole, Remove, UpdateRole, ListUsersByCompany). “Owner” for a company is either `companies.owner_id = user_id` or UCR with Owner role.

**Freelance dispatchers** are **not** in `company_users` or `user_company_roles`; they live only in **`freelance_dispatchers`** and have no company link in code.

---

## 3. Invitations

- **Exists:** Invitation system for **company users** (invite to company with a role). Repo: `internal/companytz/invitations_repo.go` (Create, GetByToken, Delete). Table **`invitations`**: token, company_id, role_id, email, invited_by, expires_at.
- **Create:** POST `/v1/companies/:companyId/invitations` (RequireAppUser) — body: email, role_id. RBAC: only roles that CanInvite can invite (e.g. Owner can invite any role). Returns `invite_link` (e.g. `https://<host>/accept-invite?token=...`).
- **Accept:** POST `/v1/invitations/accept` (RequireAppUser) — body: token. Adds current user to company via **user_company_roles** with invited role; deletes invitation.
- **Decline:** No explicit “decline” endpoint; user simply does not call accept. Invitation can expire (e.g. 7 days).
- **Link to chat:** Invitations are **not** sent via chat. They are email-based links and a separate API; chat is per user-pair (see §7).
- **Who can be invited:** Only **company_users** (users who registered via company-users auth). No “invite freelance dispatcher to company” flow; no link between `freelance_dispatchers` and invitations.

---

## 4. Cargo (груз)

- **Where:** Handlers `internal/server/handlers/cargo.go`; repo `internal/cargo/repo.go`; schema `internal/infra/cargo_schema.go`. Routes are under **`/api`** (not under `/v1/dispatchers`): `/api/cargo` (POST, GET), `/api/cargo/:id` (GET, PUT, DELETE), `/api/cargo/:id/status` (PATCH), `/api/cargo/:id/offers` (POST, GET), `/api/offers/:id/accept` (POST).
- **Auth for /api/cargo:** Only **RequireBaseHeaders** (X-Client-Token, X-Device-Type, X-Language). No JWT required. If **X-User-Token** is present, the handler uses it to set **created_by_type** and **created_by_id** (admin or dispatcher) when creating cargo; if not, and `company_id` is provided in body, created_by is set to “company” and that company_id.
- **Create:** POST `/api/cargo` — body includes title, weight, route_points, payment, truck_type, etc.; optional **company_id**. Created_by: from JWT (admin/dispatcher) or from company_id.
- **Update:** PUT `/api/cargo/:id` — blocked for route/payment after status is assigned/in_transit/delivered (`ErrCannotEditAfterAssigned`).
- **List:** GET `/api/cargo` — filter by status, weight_min/max, truck_type, created_from/to, search, with_offers, page, limit, sort.
- **Statuses (cargo):** `internal/cargo/model.go`: **created**, **searching**, **assigned**, **in_transit**, **delivered**, **cancelled**. Transitions enforced in repo `SetStatus`.
- **Who can create:** Anyone who can call the API (no role check). Creator is **recorded** as admin, dispatcher, or company from JWT/body; no restriction that only a “dispatcher” or company can create.

---

## 5. Offers / ставки

- **Entity:** Table **`offers`** (id, cargo_id, carrier_id, price, currency, comment, status, created_at). Model: `cargo.Offer`; statuses: **pending**, **accepted**, **rejected** (in code and schema).
- **Create:** POST `/api/cargo/:id/offers` — body: carrier_id (UUID), price, currency, comment. Creates offer with status **pending**. No check that caller is cargo owner or dispatcher; **/api** has no JWT.
- **List:** GET `/api/cargo/:id/offers` — returns offers for that cargo.
- **Accept:** POST `/api/offers/:id/accept`. Repo: `cargo.AcceptOffer` — sets offer status to **accepted**, cargo status to **assigned**, and other pending offers for that cargo to **rejected**. **No trip/рейс is created**; only cargo status and offer statuses change.

---

## 6. Trips / рейсы

- **No trip entity** in the codebase. There is no “trip”, “flight”, or “route” table or model for a concrete execution (рейс). Accept offer only updates cargo and offers.
- **Missing:** Trip creation on “accept offer”, trip statuses (e.g. pending_driver, assigned, loading, en_route, unloading, completed), “assign driver” (dispatcher chooses driver, driver confirms). All of this would need to be implemented.

---

## 7. Chat

- **Model:** `internal/chat/model.go` — **Conversation** (user_a_id, user_b_id, ordered so user_a_id < user_b_id), **Message** (conversation_id, sender_id, body, created_at, updated_at, deleted_at). Schema: `internal/infra/chat_schema.go` — `chat_conversations`, `chat_messages`.
- **Semantics:** Chat is **per user-pair** (one conversation per two users), not per-cargo or per-trip. No cargo_id or trip_id on conversations/messages.
- **Auth:** `RequireChatUser(jwtm)` — accepts X-User-Token (JWT) or X-User-ID (Swagger) or query `user_id` / `token` (e.g. WebSocket). Sets CtxUserID and CtxUserRole (driver | dispatcher | admin | user).
- **Endpoints (v1):** GET/POST `/v1/chat/conversations`, GET `/v1/chat/conversations/:id/messages`, POST `/v1/chat/conversations/:id/messages`, PATCH/DELETE `/v1/chat/messages/:id`, GET `/v1/chat/presence/:user_id`, GET `/v1/chat/ws` (WebSocket).
- **Invitations:** Invitations are **not** sent via chat; they are a separate API (email + token link). No “invitation sent via chat” flow.

---

## 8. Drivers

- **Representation:** Table **`drivers`** (id, phone, name, driver_type, company_id, freelancer_id, registration_step, status, etc.). Repo: `internal/drivers/repo.go`; model: `internal/drivers/model.go`. Driver app users authenticate via `/v1/auth/phone` + OTP and `/v1/registration/start` (role **driver** in JWT).
- **driver_type:** company | freelancer | driver (see `internal/domain/enums.go` and registration TransportType). Optional **company_id** and **freelancer_id** on driver record.
- **Invite driver to company:** There is **no** “invite driver to company” or “driver accepts invitation” API. Invitations in the codebase are for **company_users** (email + role). No link from invitations to drivers or to chat.
- **Company staff (dispatchers) vs drivers:** Company-side “dispatchers” are **company_users** with Dispatcher/TopDispatcher in **user_company_roles**. Freelance dispatchers are **freelance_dispatchers** and are not linked to companies.

---

## 9. Freelance dispatcher specific

- **Table:** **`freelance_dispatchers`** (and `deleted_freelance_dispatchers` for soft-delete). Model: `internal/dispatchers/model.go` (id, name, phone, password, passport, pinfl, cargo_id, driver_id, rating, work_status, etc.). Repo: `internal/dispatchers/repo.go`.
- **No “freelance_dispatchers” link to companies:** There is no table or code that links a freelance dispatcher to a company (no dispatcher_company or company_id on freelance_dispatchers). So no “list my companies” or “select company” for a freelance dispatcher.
- **No branching “if account linked to company → company token”:** Login always issues the same JWT (role=dispatcher, user_id=dispatcher id). **Company-scoped token** (access token with company_id) exists only for **app users** (role=user) after POST `/v1/auth/switch-company`; RequireAppUser reads company_id from JWT and sets CtxAppUserCompanyID. Freelance dispatchers never get company_id in the token and have no switch-company flow.
- **List my companies / select company:** **Missing** for freelance dispatchers. “List my companies” (GET `/v1/auth/companies`) and “Switch company” (POST `/v1/auth/switch-company`) exist only for **app users** (company_users) under RequireAppUser.

---

## Deliverables

### A. List of existing APIs relevant to this flow

| Method | Path | Purpose |
|--------|------|---------|
| POST | /v1/dispatchers/auth/phone | Send OTP for dispatcher auth |
| POST | /v1/dispatchers/auth/otp/verify | Verify OTP → login or register session |
| POST | /v1/dispatchers/auth/login/password | Dispatcher login by phone + password |
| POST | /v1/dispatchers/auth/reset-password/request | Request password reset |
| POST | /v1/dispatchers/auth/reset-password/confirm | Confirm reset with OTP |
| POST | /v1/dispatchers/auth/logout | Logout (invalidate refresh token) |
| POST | /v1/dispatchers/registration/complete | Complete dispatcher registration (name, password, passport, pinfl) |
| POST | /v1/auth/refresh | Refresh access token (works for driver/dispatcher/admin/user) |
| GET | /v1/dispatchers/profile | Get dispatcher profile (RequireDispatcher) |
| PATCH | /v1/dispatchers/profile | Update dispatcher profile |
| POST | /v1/dispatchers/profile/photo | Upload photo |
| GET | /v1/dispatchers/profile/photo | Get photo |
| PUT | /v1/dispatchers/profile/password | Change password |
| POST | /v1/dispatchers/profile/phone-change/request | Request phone change OTP |
| POST | /v1/dispatchers/profile/phone-change/verify | Verify phone change |
| DELETE | /v1/dispatchers/profile | Delete dispatcher account |
| GET | /v1/reference/dispatchers | Reference (e.g. work status) |
| POST | /api/cargo | Create cargo (optional X-User-Token for created_by) |
| GET | /api/cargo | List cargo (filter status, weight, etc.) |
| GET | /api/cargo/:id | Get cargo by ID |
| PUT | /api/cargo/:id | Update cargo |
| DELETE | /api/cargo/:id | Soft-delete cargo |
| PATCH | /api/cargo/:id/status | Set cargo status |
| POST | /api/cargo/:id/offers | Create offer for cargo |
| GET | /api/cargo/:id/offers | List offers for cargo |
| POST | /api/offers/:id/accept | Accept offer (cargo → assigned; no trip) |
| GET | /v1/chat/conversations | List conversations (RequireChatUser) |
| POST | /v1/chat/conversations | Get or create conversation |
| GET | /v1/chat/conversations/:id/messages | List messages |
| POST | /v1/chat/conversations/:id/messages | Send message |
| PATCH | /v1/chat/messages/:id | Edit message |
| DELETE | /v1/chat/messages/:id | Delete message |
| GET | /v1/chat/presence/:user_id | Presence |
| GET | /v1/chat/ws | WebSocket |
| POST | /v1/company-users/auth/phone | Company user: send OTP |
| POST | /v1/company-users/auth/otp/verify | Company user: verify OTP |
| POST | /v1/company-users/registration/complete | Company user: complete registration |
| GET | /v1/auth/companies | List my companies (RequireAppUser) |
| POST | /v1/auth/switch-company | Switch company → new token with company_id (RequireAppUser) |
| POST | /v1/companies | Create company (RequireAppUser, TZ “create own”) |
| POST | /v1/companies/:companyId/invitations | Create invitation (RequireAppUser) |
| POST | /v1/invitations/accept | Accept invitation (RequireAppUser) |
| GET | /v1/companies/:companyId/users | List company users (RequireAppUser) |
| PUT | /v1/companies/:companyId/users/:userId/role | Update user role (RequireAppUser) |
| DELETE | /v1/companies/:companyId/users/:userId | Remove user (RequireAppUser) |
| GET | /v1/reference/company | Company reference (roles from DB) |
| POST | /v1/admin/companies | Admin: create company |
| PATCH | /v1/admin/companies/:id/owner | Admin: set owner (company_users.id) |
| GET | /v1/admin/company-users/owners/search | Admin: search owners |

### B. Missing or incomplete pieces for the freelance dispatcher flow

1. **Dispatcher ↔ company link**  
   - No way for a freelance dispatcher to be “linked” to a company (no table, no API).  
   - No “list my companies” or “select company” for dispatchers.  
   - No “company token” or company-scoped access for dispatchers.

2. **Cargo API auth and scope**  
   - `/api/cargo` does not require JWT; only base headers. So no access control by role or company.  
   - No “list only my company’s cargo” or “only dispatchers/company can create” enforcement.  
   - Optional: require dispatcher or app-user JWT for create/update and scope list by company_id when token has company.

3. **Offers**  
   - No check that the caller is allowed to create/accept offers (e.g. cargo owner or dispatcher).  
   - **carrier_id** is free-form UUID; no check that it is a valid company or driver.  
   - Accept offer does not create a trip or assign a driver.

4. **Trips / рейсы**  
   - No trip entity: no table, no statuses (pending_driver, assigned, loading, en_route, unloading, completed), no “assign driver” or “driver confirms” flow.  
   - Needs: trip model and schema, creation on accept offer (or separate “create trip”), status transitions, and APIs for dispatcher and driver.

5. **Drivers and companies**  
   - No “invite driver to company” or “driver accepts company invitation” API.  
   - Driver has company_id/freelancer_id on record but no invitation flow linking drivers to companies.

6. **Invitations**  
   - Invitations are for company_users only (by email). No “invite by phone” or “invite dispatcher/driver” that would link freelance_dispatchers or drivers to a company.

7. **Dispatcher refresh**  
   - No dedicated POST `/v1/dispatchers/auth/refresh`; dispatchers can use POST `/v1/auth/refresh`. Optional: add a dispatcher-specific refresh route for clarity/Swagger.

8. **Swagger/OpenAPI**  
   - Document dispatcher auth (and, if added, company-scoped dispatcher and trip APIs) in `docs/openapi.yaml` so the freelance dispatcher flow is clearly described.

---

### C. Company token (company-scoped access)

**Does it exist?**  
Yes, but **only for app users** (company_users). After POST `/v1/auth/switch-company`, the backend issues a new access token with **`company_id`** in the JWT (`IssueWithCompany("user", userID, companyID)`). Middleware **RequireAppUser** uses **ParseAccessWithCompany** and sets **CtxAppUserCompanyID**. So “company token” = access token that carries an optional **company_id** claim used for scoping (e.g. list companies, switch context). It does **not** exist for freelance dispatchers; they always get a token with only role=dispatcher and user_id.

**Minimal way to add it for freelance dispatchers:**  
(1) **Link dispatcher to companies:** e.g. a table `dispatcher_company_roles` (dispatcher_id, company_id, role) or reuse a notion of “dispatcher member of company” and a way to “list my companies” for a dispatcher.  
(2) **Issue company-scoped token:** After “select company” (new endpoint, e.g. POST `/v1/dispatchers/auth/switch-company`), call **IssueWithCompany("dispatcher", dispatcherID, companyID)** and return the new access token.  
(3) **Middleware:** Add or extend a middleware that accepts dispatcher JWT and, if **company_id** is present in claims, sets a context value (e.g. CtxDispatcherCompanyID) so cargo/trip/offer handlers can scope by company.  
(4) **Optional:** Separate token type is not strictly necessary; the same JWT with an optional **company_id** claim (as today for “user”) is enough. If you need a distinct “company-only” token for stricter checks, you could add a claim like `scope: "company"` and validate it in middleware.

---

*End of report.*
