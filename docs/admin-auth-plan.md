# Admin Authentication — Implementation Plan

## Goal

Add username/password authentication with JWT sessions to the Admin API and Admin UI. This secures all gateway management endpoints behind login, with user management for admins and a first-run setup flow.

## Scope

### What's changing
- Admin API: auth middleware, user model, auth endpoints, password hashing
- Admin UI: login page, setup page, auth context, protected routes, settings pages
- Database: new `admin_users` table
- Docker: env vars for initial admin seeding

### What's NOT changing
- Gateway consumer auth (api-key-auth, jwt-auth plugins) — separate auth domain
- Gateway proxy behavior
- Existing CRUD endpoints (same logic, just protected)

## Database

```sql
admin_users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username VARCHAR(100) UNIQUE NOT NULL,
  email VARCHAR(255),
  password_hash VARCHAR(255) NOT NULL,
  role VARCHAR(20) DEFAULT 'admin' CHECK (role IN ('admin', 'viewer')),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
)
```

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/auth/status` | None | Returns `{ setup_required: bool }` |
| POST | `/auth/setup` | None (disabled after first user) | Create initial admin |
| POST | `/auth/login` | None | Returns access + refresh JWT |
| POST | `/auth/refresh` | Refresh token | Returns new access token |
| GET | `/auth/me` | Access token | Current user profile |
| PUT | `/auth/me` | Access token | Update own password/email |
| GET | `/auth/users` | Admin only | List all users |
| POST | `/auth/users` | Admin only | Create user |
| PUT | `/auth/users/:id` | Admin only | Update user |
| DELETE | `/auth/users/:id` | Admin only | Delete user (can't delete self) |

All existing endpoints (`/services`, `/routes`, `/consumers`, `/plugins`) require a valid access token. `/health` remains public.

## JWT Design

- **Access token**: 15 min expiry, contains `{ sub: user_id, username, role, exp }`
- **Refresh token**: 7 day expiry, contains `{ sub: user_id, type: "refresh", exp }`
- **Secret**: `JWT_SECRET` env var (required in production, auto-generated in development)
- **Algorithm**: HS256

## Security

- **Password hashing**: bcrypt with salt
- **Password validation**: min 8 chars, at least 1 letter + 1 number
- **Login rate limiting**: 5 attempts per IP per minute, 15 min lockout
- **Setup endpoint**: permanently disabled once any user exists (server-side check)
- **Refresh flow**: access token expires → UI uses refresh token to get new one → if refresh expired → redirect to login

## UI Pages

| Route | Page | Purpose |
|-------|------|---------|
| `/setup` | SetupPage | First-run admin creation (redirects to login if users exist) |
| `/login` | LoginPage | Username + password form |
| `/settings/users` | UsersPage | List, create, delete admin users |
| `/settings/profile` | ProfilePage | Change own password/email |

## Implementation Phases

### Phase 1: Backend auth (admin-api)
- [ ] Add `admin_users` table to schema.sql
- [ ] Add User model + schemas (models.py, schemas.py)
- [ ] Add password utilities (hashing, validation)
- [ ] Add JWT utilities (create, verify, refresh)
- [ ] Add auth router (`/auth/*` endpoints)
- [ ] Add auth middleware (protect all existing routes)
- [ ] Add env var seeding (ADMIN_USER, ADMIN_PASSWORD)
- [ ] Add login rate limiting

### Phase 2: Frontend auth (admin-ui)
- [ ] Add auth API client + types
- [ ] Add auth context (AuthProvider, useAuth hook)
- [ ] Build login page (with frontend-design agent)
- [ ] Build setup page (with frontend-design agent)
- [ ] Add protected route wrapper
- [ ] Update API client to attach JWT + handle 401
- [ ] Add token refresh logic

### Phase 3: User management UI
- [ ] Add Settings section to sidebar
- [ ] Build users list page
- [ ] Build create user form
- [ ] Build profile/change password page
- [ ] Role-based UI (hide admin-only actions for viewers)

### Phase 4: Testing + polish
- [ ] Unit tests for password utils, JWT utils
- [ ] Unit tests for auth middleware
- [ ] E2E tests for login flow
- [ ] E2E tests for setup flow
- [ ] Update existing e2e tests to handle auth
- [ ] Update docker-compose with JWT_SECRET
- [ ] Update CHANGELOG, README
