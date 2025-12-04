import os

import pytest


@pytest.fixture(scope="session")
def base_url():
    return os.environ.get("GATEWAY_URL", "http://localhost:8000")
