from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # Database
    database_url: str = "postgresql+asyncpg://flash:flash1234@localhost:5432/flash_deals"

    # JWT
    jwt_secret_key: str = "your-secret-key-change-in-production"
    jwt_algorithm: str = "HS256"
    access_token_expire_minutes: int = 60
    refresh_token_expire_days: int = 7

    # OpenTelemetry
    otel_enabled: bool = False
    otel_service_name: str = "auth-service"
    otel_exporter_otlp_endpoint: str = "http://localhost:4317"

    # Service settings
    debug: bool = False

    class Config:
        env_prefix = ""
        env_file = ".env"
        env_file_encoding = "utf-8"
        extra = "ignore"


settings = Settings()
