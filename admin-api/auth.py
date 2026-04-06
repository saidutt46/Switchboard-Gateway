"""Authentication utilities — password hashing, JWT tokens, dependencies."""

import os
import re
import time
import logging
from datetime import datetime, timedelta, timezone
from typing import Optional
from collections import defaultdict

import bcrypt
import jwt
from fastapi import Depends, HTTPException, Request
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from sqlalchemy.orm import Session

from database import get_db
from models import AdminUser

logger = logging.getLogger(__name__)

# JWT configuration
JWT_SECRET = os.getenv("JWT_SECRET", "switchboard-dev-secret-change-in-production")
JWT_ALGORITHM = "HS256"
ACCESS_TOKEN_EXPIRE_MINUTES = 15
REFRESH_TOKEN_EXPIRE_DAYS = 7

# Rate limiting for login
LOGIN_ATTEMPTS: dict[str, list[float]] = defaultdict(list)
MAX_LOGIN_ATTEMPTS = 5
LOGIN_WINDOW_SECONDS = 60
LOGIN_LOCKOUT_SECONDS = 900  # 15 minutes


# ============================================================================
# Password utilities
# ============================================================================

def hash_password(password: str) -> str:
    """Hash a password with bcrypt."""
    return bcrypt.hashpw(password.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")


def verify_password(password: str, password_hash: str) -> bool:
    """Verify a password against a bcrypt hash."""
    return bcrypt.checkpw(password.encode("utf-8"), password_hash.encode("utf-8"))


def validate_password(password: str) -> Optional[str]:
    """Validate password strength. Returns error message or None."""
    if len(password) < 8:
        return "Password must be at least 8 characters"
    if not re.search(r"[a-zA-Z]", password):
        return "Password must contain at least one letter"
    if not re.search(r"[0-9]", password):
        return "Password must contain at least one number"
    return None


# ============================================================================
# JWT utilities
# ============================================================================

def create_access_token(user_id: str, username: str, role: str) -> str:
    """Create a JWT access token."""
    payload = {
        "sub": user_id,
        "username": username,
        "role": role,
        "type": "access",
        "exp": datetime.now(timezone.utc) + timedelta(minutes=ACCESS_TOKEN_EXPIRE_MINUTES),
        "iat": datetime.now(timezone.utc),
    }
    return jwt.encode(payload, JWT_SECRET, algorithm=JWT_ALGORITHM)


def create_refresh_token(user_id: str) -> str:
    """Create a JWT refresh token."""
    payload = {
        "sub": user_id,
        "type": "refresh",
        "exp": datetime.now(timezone.utc) + timedelta(days=REFRESH_TOKEN_EXPIRE_DAYS),
        "iat": datetime.now(timezone.utc),
    }
    return jwt.encode(payload, JWT_SECRET, algorithm=JWT_ALGORITHM)


def decode_token(token: str) -> dict:
    """Decode and verify a JWT token. Raises HTTPException on failure."""
    try:
        payload = jwt.decode(token, JWT_SECRET, algorithms=[JWT_ALGORITHM])
        return payload
    except jwt.ExpiredSignatureError:
        raise HTTPException(status_code=401, detail="Token has expired")
    except jwt.InvalidTokenError:
        raise HTTPException(status_code=401, detail="Invalid token")


# ============================================================================
# Login rate limiting
# ============================================================================

def check_login_rate_limit(client_ip: str) -> None:
    """Check if login attempts are rate limited. Raises HTTPException if locked out."""
    now = time.time()
    attempts = LOGIN_ATTEMPTS[client_ip]

    # Clean old attempts
    LOGIN_ATTEMPTS[client_ip] = [t for t in attempts if now - t < LOGIN_LOCKOUT_SECONDS]
    attempts = LOGIN_ATTEMPTS[client_ip]

    # Check lockout (too many recent attempts)
    recent = [t for t in attempts if now - t < LOGIN_WINDOW_SECONDS]
    if len(recent) >= MAX_LOGIN_ATTEMPTS:
        retry_after = int(LOGIN_LOCKOUT_SECONDS - (now - attempts[-MAX_LOGIN_ATTEMPTS]))
        raise HTTPException(
            status_code=429,
            detail=f"Too many login attempts. Try again in {retry_after} seconds.",
            headers={"Retry-After": str(retry_after)},
        )


def record_login_attempt(client_ip: str) -> None:
    """Record a failed login attempt."""
    LOGIN_ATTEMPTS[client_ip].append(time.time())


# ============================================================================
# FastAPI dependencies
# ============================================================================

security = HTTPBearer(auto_error=False)


def get_current_user(
    request: Request,
    credentials: Optional[HTTPAuthorizationCredentials] = Depends(security),
    db: Session = Depends(get_db),
) -> AdminUser:
    """Dependency: extract and validate the current user from JWT."""
    if not credentials:
        raise HTTPException(status_code=401, detail="Authentication required")

    payload = decode_token(credentials.credentials)

    if payload.get("type") != "access":
        raise HTTPException(status_code=401, detail="Invalid token type")

    user_id = payload.get("sub")
    if not user_id:
        raise HTTPException(status_code=401, detail="Invalid token payload")

    user = db.query(AdminUser).filter(AdminUser.id == user_id).first()
    if not user:
        raise HTTPException(status_code=401, detail="User not found")

    return user


def require_admin(current_user: AdminUser = Depends(get_current_user)) -> AdminUser:
    """Dependency: require admin role."""
    if current_user.role != "admin":
        raise HTTPException(status_code=403, detail="Admin access required")
    return current_user


def require_write_access(current_user: AdminUser = Depends(get_current_user)) -> AdminUser:
    """Dependency: require write access (admin role). Viewers get 403 on mutations."""
    if current_user.role == "viewer":
        raise HTTPException(status_code=403, detail="Write access requires admin role")
    return current_user
