"""Auth router — login, setup, user management."""

import os
import logging
from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException, Request
from sqlalchemy.orm import Session

from database import get_db
from models import AdminUser
from schemas import (
    AuthSetup, AuthLogin, AuthTokens, AuthRefresh,
    AdminUserCreate, AdminUserUpdate, AdminUserResponse,
)
from auth import (
    hash_password, verify_password, validate_password,
    create_access_token, create_refresh_token, decode_token,
    check_login_rate_limit, record_login_attempt,
    get_current_user, require_admin,
    ACCESS_TOKEN_EXPIRE_MINUTES,
)

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/auth", tags=["Auth"])


# ============================================================================
# Public endpoints (no auth required)
# ============================================================================

@router.get("/status")
def auth_status(db: Session = Depends(get_db)):
    """Check if initial setup is required."""
    user_count = db.query(AdminUser).count()
    return {"setup_required": user_count == 0}


@router.post("/setup", response_model=AdminUserResponse, status_code=201)
def setup_initial_admin(data: AuthSetup, db: Session = Depends(get_db)):
    """Create the initial admin user. Only works when no users exist."""
    user_count = db.query(AdminUser).count()
    if user_count > 0:
        raise HTTPException(status_code=403, detail="Setup already completed. Admin user exists.")

    # Validate password
    password_error = validate_password(data.password)
    if password_error:
        raise HTTPException(status_code=400, detail=password_error)

    user = AdminUser(
        username=data.username,
        email=data.email,
        password_hash=hash_password(data.password),
        role="admin",
    )
    db.add(user)
    db.commit()
    db.refresh(user)

    logger.info("Initial admin user created", extra={"username": data.username})
    return user


@router.post("/login", response_model=AuthTokens)
def login(data: AuthLogin, request: Request, db: Session = Depends(get_db)):
    """Authenticate and return JWT tokens."""
    client_ip = request.client.host if request.client else "unknown"

    # Rate limit check
    check_login_rate_limit(client_ip)

    # Find user
    user = db.query(AdminUser).filter(AdminUser.username == data.username).first()
    if not user or not verify_password(data.password, user.password_hash):
        record_login_attempt(client_ip)
        logger.warning("Failed login attempt", extra={"username": data.username, "ip": client_ip})
        raise HTTPException(status_code=401, detail="Invalid username or password")

    # Create tokens
    access_token = create_access_token(str(user.id), user.username, user.role)
    refresh_token = create_refresh_token(str(user.id))

    logger.info("User logged in", extra={"username": user.username, "role": user.role})
    return AuthTokens(
        access_token=access_token,
        refresh_token=refresh_token,
        expires_in=ACCESS_TOKEN_EXPIRE_MINUTES * 60,
    )


@router.post("/refresh", response_model=AuthTokens)
def refresh_token(data: AuthRefresh, db: Session = Depends(get_db)):
    """Refresh an access token using a refresh token."""
    payload = decode_token(data.refresh_token)

    if payload.get("type") != "refresh":
        raise HTTPException(status_code=401, detail="Invalid token type")

    user_id = payload.get("sub")
    user = db.query(AdminUser).filter(AdminUser.id == user_id).first()
    if not user:
        raise HTTPException(status_code=401, detail="User not found")

    access_token = create_access_token(str(user.id), user.username, user.role)
    refresh_token = create_refresh_token(str(user.id))

    return AuthTokens(
        access_token=access_token,
        refresh_token=refresh_token,
        expires_in=ACCESS_TOKEN_EXPIRE_MINUTES * 60,
    )


# ============================================================================
# Authenticated endpoints
# ============================================================================

@router.get("/me", response_model=AdminUserResponse)
def get_me(current_user: AdminUser = Depends(get_current_user)):
    """Get current user profile."""
    return current_user


@router.put("/me", response_model=AdminUserResponse)
def update_me(data: AdminUserUpdate, db: Session = Depends(get_db), current_user: AdminUser = Depends(get_current_user)):
    """Update current user's own profile (email, password)."""
    if data.email is not None:
        current_user.email = data.email

    if data.password is not None:
        password_error = validate_password(data.password)
        if password_error:
            raise HTTPException(status_code=400, detail=password_error)
        current_user.password_hash = hash_password(data.password)

    # Users can't change their own role
    db.commit()
    db.refresh(current_user)
    return current_user


# ============================================================================
# Admin-only user management
# ============================================================================

@router.get("/users", response_model=list[AdminUserResponse])
def list_users(db: Session = Depends(get_db), _admin: AdminUser = Depends(require_admin)):
    """List all admin users. Admin only."""
    return db.query(AdminUser).order_by(AdminUser.created_at).all()


@router.post("/users", response_model=AdminUserResponse, status_code=201)
def create_user(data: AdminUserCreate, db: Session = Depends(get_db), _admin: AdminUser = Depends(require_admin)):
    """Create a new admin user. Admin only."""
    # Check duplicate username
    existing = db.query(AdminUser).filter(AdminUser.username == data.username).first()
    if existing:
        raise HTTPException(status_code=409, detail="Username already exists")

    # Validate password
    password_error = validate_password(data.password)
    if password_error:
        raise HTTPException(status_code=400, detail=password_error)

    user = AdminUser(
        username=data.username,
        email=data.email,
        password_hash=hash_password(data.password),
        role=data.role,
    )
    db.add(user)
    db.commit()
    db.refresh(user)

    logger.info("Admin user created", extra={"username": data.username, "role": data.role})
    return user


@router.put("/users/{user_id}", response_model=AdminUserResponse)
def update_user(user_id: UUID, data: AdminUserUpdate, db: Session = Depends(get_db), _admin: AdminUser = Depends(require_admin)):
    """Update an admin user. Admin only."""
    user = db.query(AdminUser).filter(AdminUser.id == user_id).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")

    if data.email is not None:
        user.email = data.email
    if data.role is not None and data.role != user.role:
        # Prevent demoting the last admin
        if user.role == "admin" and data.role != "admin":
            admin_count = db.query(AdminUser).filter(AdminUser.role == "admin").count()
            if admin_count <= 1:
                raise HTTPException(status_code=400, detail="Cannot demote the last admin user")
        user.role = data.role
    if data.password is not None:
        password_error = validate_password(data.password)
        if password_error:
            raise HTTPException(status_code=400, detail=password_error)
        user.password_hash = hash_password(data.password)

    db.commit()
    db.refresh(user)

    logger.info("Admin user updated", extra={"user_id": str(user_id)})
    return user


@router.delete("/users/{user_id}", status_code=204)
def delete_user(user_id: UUID, db: Session = Depends(get_db), admin: AdminUser = Depends(require_admin)):
    """Delete an admin user. Admin only. Cannot delete yourself."""
    if str(admin.id) == str(user_id):
        raise HTTPException(status_code=400, detail="Cannot delete your own account")

    user = db.query(AdminUser).filter(AdminUser.id == user_id).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")

    # Prevent deleting the last admin
    if user.role == "admin":
        admin_count = db.query(AdminUser).filter(AdminUser.role == "admin").count()
        if admin_count <= 1:
            raise HTTPException(status_code=400, detail="Cannot delete the last admin user")

    db.delete(user)
    db.commit()

    logger.info("Admin user deleted", extra={"user_id": str(user_id), "username": user.username})


def seed_admin_from_env(db: Session) -> None:
    """Create initial admin user from environment variables if no users exist."""
    admin_user = os.getenv("ADMIN_USER")
    admin_password = os.getenv("ADMIN_PASSWORD")

    if not admin_user or not admin_password:
        return

    user_count = db.query(AdminUser).count()
    if user_count > 0:
        return

    password_error = validate_password(admin_password)
    if password_error:
        logger.warning(f"ADMIN_PASSWORD validation failed: {password_error}")
        return

    user = AdminUser(
        username=admin_user,
        email=os.getenv("ADMIN_EMAIL"),
        password_hash=hash_password(admin_password),
        role="admin",
    )
    db.add(user)
    db.commit()
    logger.info(f"Initial admin user '{admin_user}' created from environment variables")
