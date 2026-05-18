from __future__ import annotations

import unittest
from types import SimpleNamespace

from fastapi import FastAPI

from runtime.observability.otel import configure_opentelemetry


class OTelTests(unittest.TestCase):
    def test_configure_opentelemetry_does_not_raise(self) -> None:
        app = FastAPI()
        settings = SimpleNamespace(
            otel_exporter_otlp_endpoint="",
            otel_service_name="test-ai",
            ai_environment="test",
        )

        configure_opentelemetry(app, settings)


if __name__ == "__main__":
    unittest.main()
