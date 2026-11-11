
from fastapi import HTTPException
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.exc import IntegrityError

import repositories
import security
from config import settings


async def register_user(
    session: AsyncSession,
    name: str,
    email: str,
    password: str
):
    existing_user = await repositories.get_user_by_email(
        session,
        email
    )

    if existing_user:
        raise HTTPException(
            status_code=409,
            detail="User with this email already exists"
        )

    password_hash = security.hash_password(password)

    try:
        user = await repositories.create_user(
            session,
            name,
            email,
            password_hash
        )

        await session.commit()

    except IntegrityError:
        await session.rollback()

        raise HTTPException(
            status_code=409,
            detail="User with this email already exists"
        )

    access_token = security.create_access_token(user["id"])

    return {
        "status": "success",
        "message": "User registered successfully",
        "user": dict(user),
        "access_token": access_token,
        "token_type": "Bearer",
        "expires_in": settings.ACCESS_TOKEN_TTL
    }


async def authenticate_user(
    session: AsyncSession,
    email: str,
    password: str
):
    user = await repositories.get_user_by_email(
        session,
        email
    )

    if not user or not security.verify_password(
        password,
        user["password_hash"]
    ):
        raise HTTPException(
            status_code=401,
            detail="Invalid email or password"
        )

    access_token = security.create_access_token(user["id"])

    return {
        "status": "success",
        "message": "Authentication successful",
        "user": {
            "id": user["id"],
            "name": user["name"],
            "email": user["email"]
        },
        "access_token": access_token,
        "token_type": "Bearer",
        "expires_in": settings.ACCESS_TOKEN_TTL
    }
