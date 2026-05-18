from __future__ import annotations

import unittest

from runtime.api.sse import EventSourceResponse


class SSETests(unittest.TestCase):
    def test_event_source_response_symbol_is_available(self) -> None:
        async def content():
            yield {"event": "ping", "data": "ok"}

        response = EventSourceResponse(content())

        self.assertEqual(response.media_type, "text/event-stream")


if __name__ == "__main__":
    unittest.main()
