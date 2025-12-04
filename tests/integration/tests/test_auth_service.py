"""
Auth Service Integration Tests

API Spec 기반 테스트 - Gateway를 통해 서비스에 접근
참조: api-spec/auth-service.v1.yaml
"""

import uuid

from playwright.sync_api import Playwright


class TestHealthCheck:
    """GET /auth/health - Auth Service health check via Gateway"""

    def test_health_check_returns_healthy(self, playwright: Playwright, base_url: str):
        """Auth Service health check - Gateway를 통해 접근"""
        api = playwright.request.new_context(base_url=base_url)
        response = api.get("/auth/health")

        assert response.status == 200
        data = response.json()
        assert data["status"] == "healthy"
        assert data["service"] == "auth-service"


class TestRegister:
    """POST /auth/register"""

    def test_register_success(self, playwright: Playwright, base_url: str):
        api = playwright.request.new_context(base_url=base_url)
        unique_email = f"test_{uuid.uuid4().hex[:8]}@example.com"

        response = api.post(
            "/auth/register",
            data={
                "email": unique_email,
                "password": "password123",
                "name": "testuser",
            },
        )

        assert response.status == 201
        data = response.json()
        assert data["email"] == unique_email
        assert data["name"] == "testuser"
        assert "id" in data
        assert "created_at" in data

    def test_register_duplicate_email_returns_409(
        self, playwright: Playwright, base_url: str
    ):
        api = playwright.request.new_context(base_url=base_url)
        unique_email = f"test_{uuid.uuid4().hex[:8]}@example.com"

        # 첫 번째 등록
        api.post(
            "/auth/register",
            data={"email": unique_email, "password": "password123", "name": "user1"},
        )

        # 중복 등록 시도
        response = api.post(
            "/auth/register",
            data={"email": unique_email, "password": "password456", "name": "user2"},
        )

        assert response.status == 409
        data = response.json()
        assert data["error"] == "EMAIL_EXISTS"

    def test_register_invalid_email_returns_422(
        self, playwright: Playwright, base_url: str
    ):
        api = playwright.request.new_context(base_url=base_url)

        response = api.post(
            "/auth/register",
            data={"email": "invalid-email", "password": "password123", "name": "user"},
        )

        assert response.status == 422

    def test_register_short_password_returns_422(
        self, playwright: Playwright, base_url: str
    ):
        api = playwright.request.new_context(base_url=base_url)

        response = api.post(
            "/auth/register",
            data={"email": "test@example.com", "password": "short", "name": "user"},
        )

        assert response.status == 422


class TestLogin:
    """POST /auth/login"""

    def test_login_success(self, playwright: Playwright, base_url: str):
        api = playwright.request.new_context(base_url=base_url)
        unique_email = f"test_{uuid.uuid4().hex[:8]}@example.com"

        # 회원가입
        api.post(
            "/auth/register",
            data={"email": unique_email, "password": "password123", "name": "user"},
        )

        # 로그인
        response = api.post(
            "/auth/login",
            data={"email": unique_email, "password": "password123"},
        )

        assert response.status == 200
        data = response.json()
        assert "access_token" in data
        assert "refresh_token" in data
        assert data["token_type"] == "Bearer"
        assert data["expires_in"] == 3600

    def test_login_wrong_password_returns_401(
        self, playwright: Playwright, base_url: str
    ):
        api = playwright.request.new_context(base_url=base_url)
        unique_email = f"test_{uuid.uuid4().hex[:8]}@example.com"

        # 회원가입
        api.post(
            "/auth/register",
            data={"email": unique_email, "password": "password123", "name": "user"},
        )

        # 잘못된 비밀번호로 로그인
        response = api.post(
            "/auth/login",
            data={"email": unique_email, "password": "wrongpassword"},
        )

        assert response.status == 401
        data = response.json()
        assert data["error"] == "INVALID_CREDENTIALS"

    def test_login_nonexistent_user_returns_401(
        self, playwright: Playwright, base_url: str
    ):
        api = playwright.request.new_context(base_url=base_url)

        response = api.post(
            "/auth/login",
            data={"email": "nonexistent@example.com", "password": "password123"},
        )

        assert response.status == 401


class TestVerify:
    """GET /auth/verify"""

    def test_verify_valid_token(self, playwright: Playwright, base_url: str):
        api = playwright.request.new_context(base_url=base_url)
        unique_email = f"test_{uuid.uuid4().hex[:8]}@example.com"

        # 회원가입 + 로그인
        api.post(
            "/auth/register",
            data={"email": unique_email, "password": "password123", "name": "user"},
        )
        login_response = api.post(
            "/auth/login",
            data={"email": unique_email, "password": "password123"},
        )
        access_token = login_response.json()["access_token"]

        # 토큰 검증
        response = api.get(
            "/auth/verify",
            headers={"Authorization": f"Bearer {access_token}"},
        )

        assert response.status == 200
        data = response.json()
        assert data["valid"] is True
        assert "user_id" in data

    def test_verify_invalid_token_returns_401(
        self, playwright: Playwright, base_url: str
    ):
        api = playwright.request.new_context(base_url=base_url)

        response = api.get(
            "/auth/verify",
            headers={"Authorization": "Bearer invalid_token"},
        )

        assert response.status == 401

    def test_verify_no_token_returns_401(self, playwright: Playwright, base_url: str):
        api = playwright.request.new_context(base_url=base_url)

        response = api.get("/auth/verify")

        assert response.status == 401


class TestRefresh:
    """POST /auth/refresh"""

    def test_refresh_token_success(self, playwright: Playwright, base_url: str):
        api = playwright.request.new_context(base_url=base_url)
        unique_email = f"test_{uuid.uuid4().hex[:8]}@example.com"

        # 회원가입 + 로그인
        api.post(
            "/auth/register",
            data={"email": unique_email, "password": "password123", "name": "user"},
        )
        login_response = api.post(
            "/auth/login",
            data={"email": unique_email, "password": "password123"},
        )
        refresh_token = login_response.json()["refresh_token"]

        # 토큰 갱신
        response = api.post(
            "/auth/refresh",
            data={"refresh_token": refresh_token},
        )

        assert response.status == 200
        data = response.json()
        assert "access_token" in data
        assert "refresh_token" in data

    def test_refresh_invalid_token_returns_401(
        self, playwright: Playwright, base_url: str
    ):
        api = playwright.request.new_context(base_url=base_url)

        response = api.post(
            "/auth/refresh",
            data={"refresh_token": "invalid_token"},
        )

        assert response.status == 401


class TestGetMe:
    """GET /auth/users/me"""

    def test_get_me_success(self, playwright: Playwright, base_url: str):
        api = playwright.request.new_context(base_url=base_url)
        unique_email = f"test_{uuid.uuid4().hex[:8]}@example.com"

        # 회원가입 + 로그인
        api.post(
            "/auth/register",
            data={"email": unique_email, "password": "password123", "name": "testuser"},
        )
        login_response = api.post(
            "/auth/login",
            data={"email": unique_email, "password": "password123"},
        )
        access_token = login_response.json()["access_token"]

        # 내 정보 조회
        response = api.get(
            "/auth/users/me",
            headers={"Authorization": f"Bearer {access_token}"},
        )

        assert response.status == 200
        data = response.json()
        assert data["email"] == unique_email
        assert data["name"] == "testuser"
        assert "id" in data
        assert "created_at" in data

    def test_get_me_no_token_returns_401(self, playwright: Playwright, base_url: str):
        api = playwright.request.new_context(base_url=base_url)

        response = api.get("/auth/users/me")

        assert response.status == 401
