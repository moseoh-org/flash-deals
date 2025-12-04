from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.instrumentation.httpx import HTTPXClientInstrumentor
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

from src.config import settings


def setup_telemetry(app):
    """OpenTelemetry 설정"""
    if not settings.otel_enabled:
        return

    # Resource 설정
    resource = Resource.create(
        {
            "service.name": settings.otel_service_name,
            "service.version": "1.0.0",
        }
    )

    # Tracer Provider 설정
    provider = TracerProvider(resource=resource)

    # OTLP Exporter 설정
    otlp_exporter = OTLPSpanExporter(
        endpoint=settings.otel_exporter_otlp_endpoint,
        insecure=True,
    )
    provider.add_span_processor(BatchSpanProcessor(otlp_exporter))

    # Global Tracer Provider 등록
    trace.set_tracer_provider(provider)

    # FastAPI 자동 계측
    FastAPIInstrumentor.instrument_app(app)

    # HTTPX 자동 계측 (프록시 요청 추적)
    HTTPXClientInstrumentor().instrument()
