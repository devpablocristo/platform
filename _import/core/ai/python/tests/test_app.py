from __future__ import annotations

import unittest

from fastapi.testclient import TestClient

from runtime.api.app import create_app
from runtime.config.settings import APISettings


class AppTests(unittest.TestCase):
    def test_health_and_ready_endpoints(self) -> None:
        app = create_app(APISettings(service_name="core-ai", api_version="0.1.0"))
        client = TestClient(app)

        health = client.get("/healthz")
        ready = client.get("/readyz")

        self.assertEqual(health.status_code, 200)
        self.assertEqual(ready.status_code, 200)
        self.assertEqual(health.json()["service"], "core-ai")
        self.assertIn("X-Request-Id", health.headers)


if __name__ == "__main__":
    unittest.main()
