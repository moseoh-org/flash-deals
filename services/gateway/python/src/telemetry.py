import socket

from opentelemetry import metrics, trace
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.instrumentation.httpx import HTTPXClientInstrumentor
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.semconv.trace import SpanAttributes

from src.config import settings


def server_request_hook(span, scope):
    """Gateway catch-all 라우트의 실제 경로를 span에 설정"""
    if span and span.is_recording():
        path = scope.get("path", "")
        if path and path != "/health":
            span.set_attribute(SpanAttributes.HTTP_ROUTE, path)
            span.update_name(f"{scope.get('method', 'HTTP')} {path}")


def setup_telemetry(app):
    """OpenTelemetry 설정 (Traces + Metrics)"""
    if not settings.otel_enabled:
        return

    # Resource 설정 (대시보드 호환성을 위한 필수 속성 포함)
    resource = Resource.create(
        {
            "service.name": settings.otel_service_name,
            "service.namespace": "flash-deals",
            "service.version": "1.0.0",
            "service.instance.id": socket.gethostname(),
            "deployment.environment.name": "development",
        }
    )

    # === Tracer Provider 설정 ===
    tracer_provider = TracerProvider(resource=resource)
    otlp_trace_exporter = OTLPSpanExporter(
        endpoint=settings.otel_exporter_otlp_endpoint,
        insecure=True,
    )
    tracer_provider.add_span_processor(BatchSpanProcessor(otlp_trace_exporter))
    trace.set_tracer_provider(tracer_provider)

    # === Meter Provider 설정 ===
    otlp_metric_exporter = OTLPMetricExporter(
        endpoint=settings.otel_exporter_otlp_endpoint,
        insecure=True,
    )
    metric_reader = PeriodicExportingMetricReader(
        otlp_metric_exporter,
        export_interval_millis=15000,  # 15초마다 메트릭 전송
    )
    meter_provider = MeterProvider(resource=resource, metric_readers=[metric_reader])
    metrics.set_meter_provider(meter_provider)

    # FastAPI 자동 계측 (트레이스 + HTTP 메트릭 생성)
    # server_request_hook: catch-all 라우트의 실제 경로를 span에 설정
    FastAPIInstrumentor.instrument_app(app, server_request_hook=server_request_hook)

    # HTTPX 자동 계측 (프록시 요청 추적)
    HTTPXClientInstrumentor().instrument()
