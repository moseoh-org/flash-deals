from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from src.auth import extract_token, public_route_validator, verify_token
from src.proxy import get_target_url, proxy_request
from src.telemetry import setup_telemetry

app = FastAPI(
    title="Flash Deals API Gateway",
    description="API Gateway for Flash Deals MSA",
    version="1.0.0",
)

# OpenTelemetry 설정
setup_telemetry(app)


@app.get("/health")
async def health_check():
    """Gateway 헬스 체크"""
    return {"status": "healthy", "service": "gateway"}


@app.api_route(
    "/{path:path}",
    methods=["GET", "POST", "PUT", "PATCH", "DELETE"],
)
async def gateway_proxy(request: Request, path: str):
    """모든 요청을 대상 서비스로 프록시"""
    full_path = f"/{path}"

    # 대상 서비스 URL 조회
    target_url = get_target_url(full_path)
    if not target_url:
        return JSONResponse(
            status_code=404,
            content={
                "error": "NOT_FOUND",
                "message": f"No service found for path: {full_path}",
            },
        )

    # JWT 검증 (공개 경로가 아닌 경우)
    if not public_route_validator.is_public(full_path, request.method):
        token = extract_token(request)
        if not token:
            return JSONResponse(
                status_code=401,
                content={
                    "error": "UNAUTHORIZED",
                    "message": "인증이 필요합니다.",
                },
            )

        verify_result = await verify_token(token)
        if not verify_result:
            return JSONResponse(
                status_code=401,
                content={
                    "error": "INVALID_TOKEN",
                    "message": "유효하지 않은 토큰입니다.",
                },
            )

    # 프록시 요청 수행
    try:
        return await proxy_request(request, target_url)
    except Exception as e:
        return JSONResponse(
            status_code=502,
            content={
                "error": "BAD_GATEWAY",
                "message": f"Failed to connect to upstream service: {str(e)}",
            },
        )
