"""
Order Service Integration Tests

API Spec 기반 테스트 - Gateway를 통해 서비스에 접근
참조: api-spec/order-service.v1.yaml
"""

import uuid

from playwright.sync_api import Playwright


class TestHealthCheck:
    """GET /orders/health - Order Service health check via Gateway"""

    def test_health_check_returns_healthy(self, playwright: Playwright, base_url: str):
        """Order Service health check - Gateway를 통해 접근"""
        api = playwright.request.new_context(base_url=base_url)
        response = api.get("/orders/health")

        assert response.status == 200
        data = response.json()
        assert data["status"] == "healthy"
        assert data["service"] == "order-service"


class TestOrderCRUD:
    """주문 CRUD 테스트"""

    def test_create_order_success(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """주문 생성 성공"""
        api = playwright.request.new_context(base_url=base_url)

        # 1. 상품 생성
        product_response = api.post(
            "/products",
            data={"name": "주문용 상품", "price": 10000, "stock": 100},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        assert product_response.status == 201
        product = product_response.json()

        # 2. 주문 생성
        response = api.post(
            "/orders",
            data={
                "items": [{"product_id": product["id"], "quantity": 2}],
                "shipping_address": {
                    "recipient_name": "홍길동",
                    "phone": "010-1234-5678",
                    "address": "서울시 강남구 테헤란로 123",
                    "address_detail": "456호",
                    "postal_code": "06234",
                },
            },
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 201
        data = response.json()
        assert data["status"] == "confirmed"
        assert data["total_amount"] == 20000  # 10000 * 2
        assert len(data["items"]) == 1
        assert data["items"][0]["quantity"] == 2
        assert data["items"][0]["product_name"] == "주문용 상품"
        assert data["shipping_address"]["recipient_name"] == "홍길동"

    def test_create_order_without_auth_returns_401(
        self, playwright: Playwright, base_url: str
    ):
        """인증 없이 주문 생성 시 401"""
        api = playwright.request.new_context(base_url=base_url)

        response = api.post(
            "/orders",
            data={
                "items": [{"product_id": str(uuid.uuid4()), "quantity": 1}],
            },
        )

        assert response.status == 401

    def test_create_order_product_not_found(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """존재하지 않는 상품 주문 시 404"""
        api = playwright.request.new_context(base_url=base_url)

        response = api.post(
            "/orders",
            data={
                "items": [{"product_id": str(uuid.uuid4()), "quantity": 1}],
            },
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 404
        data = response.json()
        assert data["error"] == "PRODUCT_NOT_FOUND"

    def test_create_order_insufficient_stock(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """재고 부족 시 400"""
        api = playwright.request.new_context(base_url=base_url)

        # 재고가 적은 상품 생성
        product_response = api.post(
            "/products",
            data={"name": "재고부족 상품", "price": 10000, "stock": 5},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        product = product_response.json()

        # 재고 초과 주문
        response = api.post(
            "/orders",
            data={
                "items": [{"product_id": product["id"], "quantity": 10}],
            },
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 400
        data = response.json()
        assert data["error"] == "INSUFFICIENT_STOCK"

    def test_get_order_success(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """주문 상세 조회"""
        api = playwright.request.new_context(base_url=base_url)

        # 상품 생성
        product_response = api.post(
            "/products",
            data={"name": "조회테스트 상품", "price": 5000, "stock": 50},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        product = product_response.json()

        # 주문 생성
        order_response = api.post(
            "/orders",
            data={"items": [{"product_id": product["id"], "quantity": 1}]},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        order = order_response.json()

        # 주문 조회
        response = api.get(
            f"/orders/{order['id']}",
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 200
        data = response.json()
        assert data["id"] == order["id"]
        assert data["total_amount"] == 5000

    def test_get_order_not_found(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """존재하지 않는 주문 조회 시 404"""
        api = playwright.request.new_context(base_url=base_url)

        fake_id = str(uuid.uuid4())
        response = api.get(
            f"/orders/{fake_id}",
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 404
        data = response.json()
        assert data["error"] == "NOT_FOUND"

    def test_list_orders(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """내 주문 목록 조회"""
        api = playwright.request.new_context(base_url=base_url)

        # 상품 생성 및 주문
        product_response = api.post(
            "/products",
            data={"name": "목록테스트 상품", "price": 3000, "stock": 100},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        product = product_response.json()

        api.post(
            "/orders",
            data={"items": [{"product_id": product["id"], "quantity": 1}]},
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        # 목록 조회
        response = api.get(
            "/orders",
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 200
        data = response.json()
        assert "items" in data
        assert "total" in data
        assert "page" in data
        assert "size" in data
        assert data["total"] >= 1


class TestOrderCancel:
    """주문 취소 테스트"""

    def test_cancel_order_success(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """주문 취소 성공"""
        api = playwright.request.new_context(base_url=base_url)

        # 상품 생성
        product_response = api.post(
            "/products",
            data={"name": "취소테스트 상품", "price": 8000, "stock": 50},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        product = product_response.json()
        initial_stock = product["stock"]

        # 주문 생성
        order_response = api.post(
            "/orders",
            data={"items": [{"product_id": product["id"], "quantity": 3}]},
            headers={"Authorization": f"Bearer {auth_token}"},
        )
        order = order_response.json()

        # 주문 취소
        response = api.post(
            f"/orders/{order['id']}/cancel",
            data={"reason": "단순 변심"},
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 200
        data = response.json()
        assert data["status"] == "cancelled"
        assert data["cancel_reason"] == "단순 변심"
        assert data["cancelled_at"] is not None

        # 재고 복구 확인
        stock_response = api.get(f"/products/{product['id']}/stock")
        stock_data = stock_response.json()
        assert stock_data["stock"] == initial_stock  # 재고가 복구되어야 함

    def test_cancel_order_not_found(
        self, playwright: Playwright, base_url: str, auth_token: str
    ):
        """존재하지 않는 주문 취소 시 404"""
        api = playwright.request.new_context(base_url=base_url)

        fake_id = str(uuid.uuid4())
        response = api.post(
            f"/orders/{fake_id}/cancel",
            headers={"Authorization": f"Bearer {auth_token}"},
        )

        assert response.status == 404
