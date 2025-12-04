from datetime import datetime
from uuid import UUID

from pydantic import BaseModel, EmailStr, Field


class RegisterRequest(BaseModel):
    email: EmailStr
    password: str = Field(min_length=8)
    name: str = Field(min_length=2, max_length=50)


class LoginRequest(BaseModel):
    email: EmailStr
    password: str


class RefreshRequest(BaseModel):
    refresh_token: str


class TokenResponse(BaseModel):
    access_token: str
    refresh_token: str
    token_type: str = "Bearer"
    expires_in: int


class VerifyResponse(BaseModel):
    valid: bool
    user_id: UUID


class UserResponse(BaseModel):
    id: UUID
    email: str
    name: str
    created_at: datetime


class ErrorResponse(BaseModel):
    error: str
    message: str
    details: dict | None = None


class HealthResponse(BaseModel):
    status: str
    service: str
