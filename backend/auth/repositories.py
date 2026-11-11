
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession


async def get_user_by_email(
    session: AsyncSession,
    email: str
):
    query = text("""
        SELECT id, name, email, password_hash
        FROM auth.users
        WHERE LOWER(email) = LOWER(:email)
        LIMIT 1
    """)

    result = await session.execute(
        query,
        {"email": email.strip()}
    )

    return result.mappings().first()


async def create_user(
    session: AsyncSession,
    name: str,
    email: str,
    password_hash: str
):
    query = text("""
        INSERT INTO auth.users (
            name,
            email,
            password_hash
        )
        VALUES (
            :name,
            :email,
            :password_hash
        )
        RETURNING id, name, email
    """)

    result = await session.execute(
        query,
        {
            "name": name.strip(),
            "email": email.strip().lower(),
            "password_hash": password_hash
        }
    )

    return result.mappings().one()
