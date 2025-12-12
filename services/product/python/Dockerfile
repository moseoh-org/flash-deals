FROM python:3.12-slim

WORKDIR /app

COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

COPY services/product/python/pyproject.toml services/product/python/uv.lock ./
RUN uv sync --frozen --no-dev

# Proto 파일 복사 및 gRPC 코드 생성
COPY api/proto/ ./proto/
RUN mkdir -p ./src/grpc_gen && \
    uv run python -m grpc_tools.protoc \
        -I./proto \
        --python_out=./src/grpc_gen \
        --grpc_python_out=./src/grpc_gen \
        ./proto/product.proto && \
    touch ./src/grpc_gen/__init__.py && \
    sed -i 's/import product_pb2/from src.grpc_gen import product_pb2/' ./src/grpc_gen/product_pb2_grpc.py

COPY services/product/python/src/ ./src/

ENV PATH="/app/.venv/bin:$PATH"

EXPOSE 8002
EXPOSE 50051

CMD ["uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8002"]
