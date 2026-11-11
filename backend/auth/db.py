
from sqlalchemy.ext.asyncio import create_async_engine, async_sessionmaker

from config import settings


if not settings.DATABASE_URL:
    raise RuntimeError("DATABASE_URL is not configured")

DATABASE_URL = settings.DATABASE_URL

if DATABASE_URL.startswith("postgresql://"):
    DATABASE_URL = DATABASE_URL.replace(
        "postgresql://",
        "postgresql+psycopg://",
        1
    )

engine = create_async_engine(
    DATABASE_URL,
    pool_pre_ping=True
)

SessionLocal = async_sessionmaker(
    bind=engine,
    expire_on_commit=False
)


async def get_db():
    async with SessionLocal() as session:
        yield session


async def close_db():
    await engine.dispose()
