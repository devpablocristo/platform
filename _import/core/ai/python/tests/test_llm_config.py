from __future__ import annotations

import unittest

from runtime.config.llm import (
    default_model_for_provider,
    normalize_provider,
    provider_requires_api_key,
    resolve_model_name,
    validate_provider_api_key,
)


class LLMConfigTests(unittest.TestCase):
    def test_resolve_model_uses_defaults(self) -> None:
        self.assertEqual(normalize_provider("  OPENAI "), "openai")
        self.assertEqual(default_model_for_provider("google_ai_studio"), "gemini-flash-latest")
        self.assertEqual(resolve_model_name("stub", None), "stub")
        self.assertEqual(resolve_model_name("openai", "gpt-4.1-mini"), "gpt-4.1-mini")

    def test_provider_requires_api_key(self) -> None:
        self.assertFalse(provider_requires_api_key("stub"))
        self.assertFalse(provider_requires_api_key("ollama"))
        self.assertTrue(provider_requires_api_key("openai"))

    def test_validate_provider_api_key(self) -> None:
        validate_provider_api_key("stub", None)
        with self.assertRaisesRegex(ValueError, "LLM_API_KEY"):
            validate_provider_api_key("openai", "", error_message="LLM_API_KEY missing")


if __name__ == "__main__":
    unittest.main()
