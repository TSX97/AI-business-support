
from datetime import datetime, timedelta, timezone

import jwt
from pwdlib import PasswordHash

from config import settings


password_hash = PasswordHash.recommended()


def hash_password(password: str) -> str:
    return password_hash.hash(password)


def verify_password(
    password: str,
    hashed_password: str
) -> bool:
    return password_hash.verify(password, hashed_password)


def create_access_token(user_id: int) -> str:
    if not settings.JWT_SECRET:
        raise RuntimeError("JWT_SECRET is not configured")

    now = datetime.now(timezone.utc)

    payload = {
        "sub": str(user_id),
        "iat": now,
        "exp": now + timedelta(
            seconds=settings.ACCESS_TOKEN_TTL
        )
    }

    return jwt.encode(
        payload,
        settings.JWT_SECRET,
        algorithm=settings.JWT_ALGORITHM
    )
