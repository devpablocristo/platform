from __future__ import annotations

import time
import unittest
from types import SimpleNamespace

from pydantic import BaseModel

from runtime.completions import (
    LLMBudgetExceededError,
    LLMCompletion,
    LLMError,
    OllamaChatClient,
    StubLLMClient,
    build_llm_client,
    validate_json_completion,
)


def _settings(**overrides):
    values = {
        "llm_provider": "stub",
        "llm_model": "stub",
        "llm_api_key": None,
        "llm_base_url": None,
        "llm_timeout_ms": 5000,
        "llm_max_retries": 3,
        "llm_max_output_tokens": 700,
        "llm_max_calls_per_request": 3,
        "llm_budget_tokens_per_request": 2500,
        "llm_rate_limit_rps": 1000.0,
    }
    values.update(overrides)
    return SimpleNamespace(**values)


class _FailingOllama(OllamaChatClient):
    def __init__(self, settings) -> None:
        super().__init__(settings)
        self.calls = 0

    def _complete_json_once(self, *, system_prompt: str, user_prompt: str) -> LLMCompletion:
        _ = (system_prompt, user_prompt)
        self.calls += 1
        raise LLMError("transport")


class _Payload(BaseModel):
    value: str


class CompletionTests(unittest.TestCase):
    def test_build_llm_client_returns_stub(self) -> None:
        client = build_llm_client(_settings())
        self.assertIsInstance(client, StubLLMClient)

    def test_stub_budget_enforcement(self) -> None:
        client = StubLLMClient(_settings(llm_max_calls_per_request=1, llm_budget_tokens_per_request=50_000))
        with client.request_scope():
            client.complete_json(system_prompt="s", user_prompt="u")
            time.sleep(0.002)
            with self.assertRaises(LLMBudgetExceededError):
                client.complete_json(system_prompt="s", user_prompt="u")

    def test_ollama_single_attempt(self) -> None:
        client = _FailingOllama(_settings(llm_provider="ollama", llm_model="llama3.1"))
        with self.assertRaises(LLMError):
            client.complete_json(system_prompt="s", user_prompt="u")
        self.assertEqual(client.calls, 1)

    def test_validate_json_completion(self) -> None:
        parsed = validate_json_completion('{"value":"ok"}', _Payload)
        self.assertEqual(parsed.value, "ok")


if __name__ == "__main__":
    unittest.main()
