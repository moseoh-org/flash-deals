from uuid import UUID

from src.config import settings
from src.database import get_connection
from src.generated.query import AsyncQuerier
from src.schemas import TokenResponse, UserResponse
from src.security import (
    create_access_token,
    create_refresh_token,
    hash_password,
    verify_password,
    verify_refresh_token,
)


class AuthServiceError(Exception):
    def __init__(self, error: str, message: str, status_code: int = 400):
        self.error = error
        self.message = message
        self.status_code = status_code
        super().__init__(message)


async def register_user(email: str, password: str, name: str) -> UserResponse:
    async with get_connection() as conn:
        querier = AsyncQuerier(conn)

        exists = await querier.exists_user_by_email(email=email)
        if exists:
            raise AuthServiceError("EMAIL_EXISTS", "이미 존재하는 이메일입니다.", 409)

        password_hash = hash_password(password)
        user = await querier.create_user(email=email, password_hash=password_hash, name=name)
        await conn.commit()

        if user is None:
            raise AuthServiceError("CREATE_FAILED", "사용자 생성에 실패했습니다.", 500)

        return UserResponse(
            id=user.id,
            email=user.email,
            name=user.name,
            created_at=user.created_at,
        )


async def login_user(email: str, password: str) -> TokenResponse:
    async with get_connection() as conn:
        querier = AsyncQuerier(conn)

        user = await querier.get_user_by_email(email=email)
        if user is None:
            raise AuthServiceError("INVALID_CREDENTIALS", "이메일 또는 비밀번호가 올바르지 않습니다.", 401)

        if not verify_password(password, user.password_hash):
            raise AuthServiceError("INVALID_CREDENTIALS", "이메일 또는 비밀번호가 올바르지 않습니다.", 401)

        access_token = create_access_token(user.id)
        refresh_token = create_refresh_token(user.id)

        return TokenResponse(
            access_token=access_token,
            refresh_token=refresh_token,
            expires_in=settings.access_token_expire_minutes * 60,
        )


async def refresh_tokens(refresh_token: str) -> TokenResponse:
    user_id = verify_refresh_token(refresh_token)
    if user_id is None:
        raise AuthServiceError("INVALID_TOKEN", "유효하지 않은 리프레시 토큰입니다.", 401)

    async with get_connection() as conn:
        querier = AsyncQuerier(conn)
        user = await querier.get_user_by_id(id=user_id)
        if user is None:
            raise AuthServiceError("USER_NOT_FOUND", "사용자를 찾을 수 없습니다.", 401)

    new_access_token = create_access_token(user_id)
    new_refresh_token = create_refresh_token(user_id)

    return TokenResponse(
        access_token=new_access_token,
        refresh_token=new_refresh_token,
        expires_in=settings.access_token_expire_minutes * 60,
    )


async def get_user_by_id(user_id: UUID) -> UserResponse:
    async with get_connection() as conn:
        querier = AsyncQuerier(conn)
        user = await querier.get_user_by_id(id=user_id)
        if user is None:
            raise AuthServiceError("USER_NOT_FOUND", "사용자를 찾을 수 없습니다.", 404)

        return UserResponse(
            id=user.id,
            email=user.email,
            name=user.name,
            created_at=user.created_at,
        )
