"""
Gateway Integration Tests

Gateway 자체 기능 테스트
"""

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
