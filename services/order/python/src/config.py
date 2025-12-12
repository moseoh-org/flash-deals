from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # App
    app_host: str = "0.0.0.0"
    app_port: int = 8003
    app_debug: bool = False

    # Database
    db_host: str = "localhost"
    db_port: int = 5432
    db_name: str = "flash_deals"
    db_user: str = "flash"
    db_password: str = "flash1234"

    @property
    def database_url(self) -> str:
        return f"postgresql+asyncpg://{self.db_user}:{self.db_password}@{self.db_host}:{self.db_port}/{self.db_name}"

    # Service URLs
    product_service_url: str = "http://localhost:8002"

    # Product Client (http, http_pool, grpc)
    product_client_type: str = "http"  # "http", "http_pool", "grpc"
    product_grpc_host: str = "product"
    product_grpc_port: int = 50051

    # OpenTelemetry
    otel_enabled: bool = False
    otel_service_name: str = "order-service"
    otel_exporter_otlp_endpoint: str = "http://localhost:4317"

    class Config:
        env_prefix = ""
        env_file = ".env"
        env_file_encoding = "utf-8"
        extra = "ignore"


settings = Settings()
