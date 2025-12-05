import os
import uuid

import pytest
import requests


@pytest.fixture(scope="session")
def base_url():
    return os.environ.get("GATEWAY_URL", "http://localhost:8000")


@pytest.fixture(scope="session")
def auth_token(base_url):
    """테스트용 인증 토큰 생성"""
    unique_email = f"test_{uuid.uuid4().hex[:8]}@example.com"

    # 회원가입
    requests.post(
        f"{base_url}/auth/register",
        json={"email": unique_email, "password": "testpassword123", "name": "testuser"},
    )

    # 로그인
    login_response = requests.post(
        f"{base_url}/auth/login",
        json={"email": unique_email, "password": "testpassword123"},
    )

    return login_response.json()["access_token"]
