# 공개 경로 (JWT 검증 없이 접근 가능)
PUBLIC_PATHS = [
    "/health",
    "/auth/register",
    "/auth/login",
    "/auth/refresh",
]

# 공개 경로 prefix (해당 prefix로 시작하는 모든 경로)
PUBLIC_PREFIXES = [
    "/docs",
    "/openapi",
]


def is_public_path(path: str) -> bool:
    """해당 경로가 공개 경로인지 확인"""
    # 정확히 일치하는 공개 경로
    if path in PUBLIC_PATHS:
        return True

    # prefix로 시작하는 공개 경로
    for prefix in PUBLIC_PREFIXES:
        if path.startswith(prefix):
            return True

    return False
