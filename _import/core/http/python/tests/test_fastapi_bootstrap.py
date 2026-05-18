from __future__ import annotations

import unittest

from fastapi import FastAPI
from fastapi.testclient import TestClient

from httpserver.fastapi_bootstrap import (
    apply_permissive_cors,
    install_request_context_middleware,
    register_common_exception_handlers,
)
import structlog


class FastAPIBootstrapTests(unittest.TestCase):
    def test_bootstrap_helpers(self) -> None:
        structlog.configure(wrapper_class=structlog.make_filtering_bound_logger(40))
        log = structlog.get_logger("httpserver_test")

        app = FastAPI()
        apply_permissive_cors(app)
        install_request_context_middleware(
            app,
            lambda _rid: None,
            lambda: None,
        )
        register_common_exception_handlers(app, log)

        @app.get("/healthz")
        async def healthz() -> dict[str, str]:
            return {"status": "ok"}

        client = TestClient(app)
        response = client.get("/healthz")
        self.assertEqual(response.status_code, 200)
        self.assertIn("X-Request-ID", response.headers)


if __name__ == "__main__":
    unittest.main()
