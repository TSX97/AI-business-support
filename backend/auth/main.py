from contextlib import asynccontextmanager

from fastapi import FastAPI

from db import close_db
from routes import router


@asynccontextmanager
async def lifespan(app: FastAPI):
    yield
    await close_db()


app = FastAPI(
    title="Auth Service",
    description="User registration and authentication service",
    version="1.0.0",
    lifespan=lifespan
)


app.include_router(router)
