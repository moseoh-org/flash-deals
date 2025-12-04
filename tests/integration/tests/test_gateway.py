"""
Gateway Integration Tests

Gateway 자체 기능 테스트
"""

import uuid

from playwright.sync_api import Playwright


class TestHealthCheck:
    """GET /health - Gateway health check"""

    def test_health_check_returns_healthy(self, playwright: Playwright, base_url: str):
        """Gateway health check"""
        api = playwright.request.new_context(base_url=base_url)
        response = api.get("/health")

        assert response.status == 200
        data = response.json()
        assert data["status"] == "healthy"
        assert data["service"] == "gateway"


class TestRouting:
    """Gateway routing tests"""

    def test_unknown_path_returns_404(self, playwright: Playwright, base_url: str):
        """알 수 없는 경로는 404 반환"""
        api = playwright.request.new_context(base_url=base_url)
        response = api.get("/unknown/path")

        assert response.status == 404
        data = response.json()
        assert data["error"] == "NOT_FOUND"


class TestJwtAuthentication:
    """Gateway JWT 인증 테스트"""

    def test_protected_route_without_token_returns_401(
        self, playwright: Playwright, base_url: str
    ):
        """인증 필요 경로에 토큰 없이 접근 시 401 반환"""
        api = playwright.request.new_context(base_url=base_url)

        # /auth/users/me는 인증 필요
        response = api.get("/auth/users/me")

        assert response.status == 401
        data = response.json()
        assert data["error"] == "UNAUTHORIZED"

    def test_protected_route_with_invalid_token_returns_401(
        self, playwright: Playwright, base_url: str
    ):
        """인증 필요 경로에 잘못된 토큰으로 접근 시 401 반환"""
        api = playwright.request.new_context(base_url=base_url)

        response = api.get(
            "/auth/users/me",
            headers={"Authorization": "Bearer invalid_token"},
        )

        assert response.status == 401
        data = response.json()
        assert data["error"] == "INVALID_TOKEN"

    def test_protected_route_with_valid_token_succeeds(
        self, playwright: Playwright, base_url: str
    ):
        """인증 필요 경로에 유효한 토큰으로 접근 시 성공"""
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

        # /auth/users/me 접근
        response = api.get(
            "/auth/users/me",
            headers={"Authorization": f"Bearer {access_token}"},
        )

        assert response.status == 200
        data = response.json()
        assert data["email"] == unique_email

    def test_public_route_without_token_succeeds(
        self, playwright: Playwright, base_url: str
    ):
        """공개 경로는 토큰 없이 접근 가능"""
        api = playwright.request.new_context(base_url=base_url)

        # /auth/health는 공개 경로
        response = api.get("/auth/health")

        assert response.status == 200

    def test_auth_verify_requires_token(
        self, playwright: Playwright, base_url: str
    ):
        """/auth/verify는 인증 필요"""
        api = playwright.request.new_context(base_url=base_url)

        response = api.get("/auth/verify")

        assert response.status == 401
        data = response.json()
        assert data["error"] == "UNAUTHORIZED"

    def test_auth_refresh_is_public(
        self, playwright: Playwright, base_url: str
    ):
        """/auth/refresh는 공개 경로 (body로 refresh_token 전달)"""
        api = playwright.request.new_context(base_url=base_url)

        # refresh_token 없이 호출 시 Gateway는 통과하지만 Auth Service에서 401
        response = api.post(
            "/auth/refresh",
            data={"refresh_token": "invalid"},
        )

        # Gateway에서 401이 아닌 Auth Service에서 401 반환
        assert response.status == 401
        data = response.json()
        assert data["error"] == "INVALID_TOKEN"
