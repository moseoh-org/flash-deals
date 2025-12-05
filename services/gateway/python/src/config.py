from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # App
    app_host: str = "0.0.0.0"
    app_port: int = 8000
    app_debug: bool = False

    # Service URLs (환경변수로 주입)
    auth_service_url: str = "http://localhost:8001"
    product_service_url: str = "http://localhost:8002"
    order_service_url: str = "http://localhost:8003"

    # OpenTelemetry
    otel_enabled: bool = False
    otel_service_name: str = "gateway"
    otel_exporter_otlp_endpoint: str = "http://localhost:4317"

    class Config:
        env_prefix = ""
        env_file = ".env"
        env_file_encoding = "utf-8"
        extra = "ignore"


settings = Settings()
