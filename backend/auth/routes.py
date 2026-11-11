
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from db import get_db
from schemas import (
    RegisterRequest,
    LoginRequest,
    AuthenticationResponse,
    HealthResponse
)
from services import register_user, authenticate_user


router = APIRouter()


@router.post(
    "/register",
    response_model=AuthenticationResponse,
    status_code=201
)
async def register(
    request: RegisterRequest,
    session: AsyncSession = Depends(get_db)
):
    return await register_user(
        session=session,
        name=request.name,
        email=request.email,
        password=request.password
    )


@router.post(
    "/login",
    response_model=AuthenticationResponse
)
async def login(
    request: LoginRequest,
    session: AsyncSession = Depends(get_db)
):
    return await authenticate_user(
        session=session,
        email=request.email,
        password=request.password
    )


@router.get(
    "/health",
    response_model=HealthResponse
)
async def health():
    return {"status": "healthy"}
