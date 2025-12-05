"""
Product Service Integration Tests

API Spec 기반 테스트 - Gateway를 통해 서비스에 접근
참조: api-spec/product-service.v1.yaml
"""

import uuid
from datetime import datetime, timedelta, timezone

from playwright.sync_api import Playwright


class TestHealthCheck:
    """GET /products/health - Product Service health check via Gateway"""

    def test_health_check_returns_healthy(self, playwright: Playwright, base_url: str):
        """Product Service health check - Gateway를 통해 접근"""
        api = playwright.request.new_context(base_url=base_url)
        response = api.get("/products/health")

        assert response.status == 200
        data = response.json()
        assert data["status"] == "healthy"
        assert data["service"] == "product-service"


class TestProductCRUD:
    """상품 CRUD 테스트"""

    def test_create_product_success(self, playwright: Playwright, base_url: str, auth_token: str):
        """상품 등록 성공"""
        api = playwright.request.new_context(base_url=base_url)

        response = api.post(
            "/products",
            data={
                "name": "테스트 상품",
                "description": "테스트 상품 설명",
                "price": 10000,
                "stock": 100,
                "category": "test",
            },
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 201
        data = response.json()
        assert data["name"] == "테스트 상품"
        assert data["price"] == 10000
        assert data["stock"] == 100
        assert "id" in data
        assert "created_at" in data

    def test_create_product_without_auth_returns_401(
        self, playwright: Playwright, base_url: str
    ):
        """인증 없이 상품 등록 시 401"""
        api = playwright.request.new_context(base_url=base_url)

        response = api.post(
            "/products",
            data={
                "name": "테스트 상품",
                "price": 10000,
                "stock": 100,
            },
        )

        assert response.status == 401

    def test_get_product_success(self, playwright: Playwright, base_url: str, auth_token: str):
        """상품 상세 조회"""
        api = playwright.request.new_context(base_url=base_url)

        # 상품 생성
        create_response = api.post(
            "/products",
            data={"name": "조회 테스트", "price": 5000, "stock": 50},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        product_id = create_response.json()["id"]

        # 상품 조회
        response = api.get(f"/products/{product_id}")

        assert response.status == 200
        data = response.json()
        assert data["id"] == product_id
        assert data["name"] == "조회 테스트"

    def test_get_product_not_found(self, playwright: Playwright, base_url: str):
        """존재하지 않는 상품 조회 시 404"""
        api = playwright.request.new_context(base_url=base_url)

        fake_id = str(uuid.uuid4())
        response = api.get(f"/products/{fake_id}")

        assert response.status == 404
        data = response.json()
        assert data["error"] == "NOT_FOUND"

    def test_list_products(self, playwright: Playwright, base_url: str, auth_token: str):
        """상품 목록 조회"""
        api = playwright.request.new_context(base_url=base_url)

        # 상품 생성
        api.post(
            "/products",
            data={"name": "목록 테스트 1", "price": 1000, "stock": 10},
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        # 목록 조회
        response = api.get("/products")

        assert response.status == 200
        data = response.json()
        assert "items" in data
        assert "total" in data
        assert "page" in data
        assert "size" in data
        assert data["total"] >= 1

    def test_list_products_with_category_filter(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """카테고리 필터로 상품 목록 조회"""
        api = playwright.request.new_context(base_url=base_url)
        unique_category = f"test_cat_{uuid.uuid4().hex[:8]}"

        # 특정 카테고리 상품 생성
        api.post(
            "/products",
            data={"name": "필터 테스트", "price": 1000, "stock": 10, "category": unique_category},
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        # 카테고리 필터로 조회
        response = api.get(f"/products?category={unique_category}")

        assert response.status == 200
        data = response.json()
        assert data["total"] >= 1
        for item in data["items"]:
            assert item["category"] == unique_category

    def test_update_product_success(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """상품 수정 성공"""
        api = playwright.request.new_context(base_url=base_url)

        # 상품 생성
        create_response = api.post(
            "/products",
            data={"name": "수정 전", "price": 1000, "stock": 10},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        product_id = create_response.json()["id"]

        # 상품 수정
        response = api.patch(
            f"/products/{product_id}",
            data={"name": "수정 후", "price": 2000},
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 200
        data = response.json()
        assert data["name"] == "수정 후"
        assert data["price"] == 2000


class TestStock:
    """재고 관리 테스트"""

    def test_get_stock(self, playwright: Playwright, base_url: str, auth_token: str):
        """재고 조회"""
        api = playwright.request.new_context(base_url=base_url)

        # 상품 생성
        create_response = api.post(
            "/products",
            data={"name": "재고 테스트", "price": 1000, "stock": 100},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        product_id = create_response.json()["id"]

        # 재고 조회
        response = api.get(f"/products/{product_id}/stock")

        assert response.status == 200
        data = response.json()
        assert data["product_id"] == product_id
        assert data["stock"] == 100

    def test_update_stock_decrease(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """재고 감소"""
        api = playwright.request.new_context(base_url=base_url)

        # 상품 생성
        create_response = api.post(
            "/products",
            data={"name": "재고 감소 테스트", "price": 1000, "stock": 100},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        product_id = create_response.json()["id"]

        # 재고 감소
        response = api.patch(
            f"/products/{product_id}/stock",
            data={"delta": -10},
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 200
        data = response.json()
        assert data["stock"] == 90

    def test_update_stock_increase(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """재고 증가"""
        api = playwright.request.new_context(base_url=base_url)

        # 상품 생성
        create_response = api.post(
            "/products",
            data={"name": "재고 증가 테스트", "price": 1000, "stock": 100},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        product_id = create_response.json()["id"]

        # 재고 증가
        response = api.patch(
            f"/products/{product_id}/stock",
            data={"delta": 50},
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 200
        data = response.json()
        assert data["stock"] == 150

    def test_update_stock_insufficient_returns_400(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """재고 부족 시 400"""
        api = playwright.request.new_context(base_url=base_url)

        # 상품 생성 (재고 10)
        create_response = api.post(
            "/products",
            data={"name": "재고 부족 테스트", "price": 1000, "stock": 10},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        product_id = create_response.json()["id"]

        # 재고 초과 감소 시도
        response = api.patch(
            f"/products/{product_id}/stock",
            data={"delta": -20},
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 400
        data = response.json()
        assert data["error"] == "INSUFFICIENT_STOCK"


class TestDeals:
    """핫딜 테스트"""

    def test_create_deal_success(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """핫딜 등록 성공"""
        api = playwright.request.new_context(base_url=base_url)

        # 상품 생성
        create_response = api.post(
            "/products",
            data={"name": "핫딜 상품", "price": 100000, "stock": 100},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        product_id = create_response.json()["id"]

        # 핫딜 등록
        now = datetime.now(timezone.utc)
        starts_at = (now - timedelta(minutes=5)).isoformat()
        ends_at = (now + timedelta(hours=1)).isoformat()

        response = api.post(
            "/products/deals",
            data={
                "product_id": product_id,
                "deal_price": 80000,
                "deal_stock": 50,
                "starts_at": starts_at,
                "ends_at": ends_at,
            },
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 201
        data = response.json()
        assert data["product_id"] == product_id
        assert data["deal_price"] == 80000
        assert data["deal_stock"] == 50
        assert data["remaining_stock"] == 50
        assert data["status"] == "active"
        assert data["original_price"] == 100000

    def test_get_deal(self, playwright: Playwright, base_url: str, auth_token: str):
        """핫딜 상세 조회"""
        api = playwright.request.new_context(base_url=base_url)

        # 상품 및 핫딜 생성
        create_response = api.post(
            "/products",
            data={"name": "핫딜 조회 상품", "price": 50000, "stock": 100},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        product_id = create_response.json()["id"]

        now = datetime.now(timezone.utc)
        deal_response = api.post(
            "/products/deals",
            data={
                "product_id": product_id,
                "deal_price": 40000,
                "deal_stock": 30,
                "starts_at": (now - timedelta(minutes=5)).isoformat(),
                "ends_at": (now + timedelta(hours=1)).isoformat(),
            },
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        deal_id = deal_response.json()["id"]

        # 핫딜 조회
        response = api.get(f"/products/deals/{deal_id}")

        assert response.status == 200
        data = response.json()
        assert data["id"] == deal_id
        assert data["product"] is not None
        assert data["product"]["name"] == "핫딜 조회 상품"

    def test_list_active_deals(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """진행 중인 핫딜 목록 조회"""
        api = playwright.request.new_context(base_url=base_url)

        # 상품 및 핫딜 생성
        create_response = api.post(
            "/products",
            data={"name": "활성 핫딜 상품", "price": 30000, "stock": 100},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        product_id = create_response.json()["id"]

        now = datetime.now(timezone.utc)
        api.post(
            "/products/deals",
            data={
                "product_id": product_id,
                "deal_price": 25000,
                "deal_stock": 20,
                "starts_at": (now - timedelta(minutes=5)).isoformat(),
                "ends_at": (now + timedelta(hours=1)).isoformat(),
            },
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        # 활성 핫딜 목록 조회
        response = api.get("/products/deals")

        assert response.status == 200
        data = response.json()
        assert "items" in data
        assert "total" in data
        assert data["total"] >= 1

    def test_deal_not_found(self, playwright: Playwright, base_url: str):
        """존재하지 않는 핫딜 조회 시 404"""
        api = playwright.request.new_context(base_url=base_url)

        fake_id = str(uuid.uuid4())
        response = api.get(f"/products/deals/{fake_id}")

        assert response.status == 404
        data = response.json()
        assert data["error"] == "NOT_FOUND"
