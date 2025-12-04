from uuid import UUID

from fastapi import Depends, FastAPI, Header
from fastapi.responses import JSONResponse

from src.schemas import (
    ErrorResponse,
    HealthResponse,
    LoginRequest,
    RefreshRequest,
    RegisterRequest,
    TokenResponse,
    UserResponse,
    VerifyResponse,
)
from src.security import verify_access_token
from src.service import AuthServiceError, get_user_by_id, login_user, refresh_tokens, register_user
from src.telemetry import setup_telemetry

app = FastAPI(
    title="Auth Service",
    description="사용자 인증 및 JWT 토큰 관리 서비스",
    version="1.0.0",
)

setup_telemetry(app)


@app.exception_handler(AuthServiceError)
async def auth_service_error_handler(request, exc: AuthServiceError):
    return JSONResponse(
        status_code=exc.status_code,
        content=ErrorResponse(error=exc.error, message=exc.message).model_dump(),
    )


def get_token_from_header(authorization: str | None = Header(default=None)) -> str | None:
    if authorization is None:
        return None
    if not authorization.startswith("Bearer "):
        return None
    return authorization[7:]


def get_current_user_id(token: str | None = Depends(get_token_from_header)) -> UUID:
    if token is None:
        raise AuthServiceError("UNAUTHORIZED", "인증이 필요합니다.", 401)
    user_id = verify_access_token(token)
    if user_id is None:
        raise AuthServiceError("INVALID_TOKEN", "유효하지 않은 토큰입니다.", 401)
    return user_id


@app.get("/health", response_model=HealthResponse)
async def health_check():
    return HealthResponse(status="healthy", service="auth-service")


@app.post("/auth/register", response_model=UserResponse, status_code=201)
async def register(request: RegisterRequest):
    return await register_user(
        email=request.email,
        password=request.password,
        name=request.name,
    )


@app.post("/auth/login", response_model=TokenResponse)
async def login(request: LoginRequest):
    return await login_user(email=request.email, password=request.password)


@app.post("/auth/refresh", response_model=TokenResponse)
async def refresh(request: RefreshRequest):
    return await refresh_tokens(refresh_token=request.refresh_token)


@app.get("/auth/verify", response_model=VerifyResponse)
async def verify(user_id: UUID = Depends(get_current_user_id)):
    return VerifyResponse(valid=True, user_id=user_id)


@app.get("/users/me", response_model=UserResponse)
async def get_me(user_id: UUID = Depends(get_current_user_id)):
    return await get_user_by_id(user_id)
