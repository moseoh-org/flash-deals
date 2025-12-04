from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

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
