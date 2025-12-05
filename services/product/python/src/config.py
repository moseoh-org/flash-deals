from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # Server
    host: str = "0.0.0.0"
    port: int = 8002

    # Database
    database_url: str = "postgresql+asyncpg://flash:flash1234@localhost:5432/flash_deals"

    # OpenTelemetry
    otel_enabled: bool = False
    otel_service_name: str = "product-service"
    otel_exporter_otlp_endpoint: str = "http://localhost:4317"

    # Service settings
    debug: bool = False

    class Config:
        env_prefix = ""
        env_file = ".env"
        env_file_encoding = "utf-8"
        extra = "ignore"


settings = Settings()
