from __future__ import annotations

import unittest

from runtime.orchestrator import OrchestratorLimits, orchestrate
from runtime.types import ChatChunk, Message, ToolCall, ToolDeclaration


class StaticProvider:
    def __init__(self, chunks: list[list[ChatChunk]]) -> None:
        self._chunks = chunks
        self._index = 0

    async def chat(self, messages, *, tools=None, temperature=None, max_tokens=None):
        del messages, tools, temperature, max_tokens
        current = self._chunks[self._index]
        self._index += 1
        for chunk in current:
            yield chunk


class OrchestratorTests(unittest.IsolatedAsyncioTestCase):
    async def test_orchestrate_runs_tool_and_emits_done(self) -> None:
        provider = StaticProvider(
            [
                [
                    ChatChunk(type="text", text="thinking"),
                    ChatChunk(type="tool_call", tool_call=ToolCall(name="sum", arguments={"a": 2, "b": 3})),
                ],
                [
                    ChatChunk(type="text", text="done"),
                ],
            ]
        )

        async def handler(**kwargs):
            return {"total": kwargs["a"] + kwargs["b"]}

        chunks = [
            chunk
            async for chunk in orchestrate(
                provider,
                [Message(role="user", content="sum please")],
                [ToolDeclaration(name="sum", description="sum numbers")],
                {"sum": handler},
            )
        ]

        self.assertEqual(chunks[0].text, "thinking")
        self.assertEqual(chunks[1].tool_call.name, "sum")
        tool_result = chunks[2]
        self.assertEqual(tool_result.type, "tool_result")
        self.assertEqual(tool_result.tool_call.arguments["total"], 5)
        self.assertEqual(chunks[-1].type, "done")

    async def test_orchestrate_times_out(self) -> None:
        provider = StaticProvider([])
        chunks = [
            chunk
            async for chunk in orchestrate(
                provider,
                [Message(role="user", content="hi")],
                [],
                {},
                limits=OrchestratorLimits(total_timeout_seconds=0.0),
            )
        ]
        self.assertEqual(chunks[0].text, "request timed out")
        self.assertEqual(chunks[-1].type, "done")


if __name__ == "__main__":
    unittest.main()
