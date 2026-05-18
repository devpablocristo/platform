from __future__ import annotations

import unittest

from runtime.logging import bind_request_context, clear_request_context, context_fields, update_request_context


class LoggingTests(unittest.TestCase):
    def test_context_fields(self) -> None:
        bind_request_context("req-1", "acme", "user-1")
        update_request_context(user_id="user-2")
        fields = context_fields()
        self.assertEqual(fields["request_id"], "req-1")
        self.assertEqual(fields["tenant_id"], "acme")
        self.assertEqual(fields["user_id"], "user-2")
        clear_request_context()


if __name__ == "__main__":
    unittest.main()
