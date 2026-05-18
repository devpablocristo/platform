from __future__ import annotations

import unittest

from runtime.provider_factory import ProviderFactory, ProviderFactoryError
from runtime.types import EchoProvider


class ProviderFactoryTests(unittest.TestCase):
    def test_register_and_build(self) -> None:
        factory = ProviderFactory()
        factory.register("echo", lambda: EchoProvider())

        provider = factory.build("echo")
        self.assertIsInstance(provider, EchoProvider)
        self.assertEqual(factory.names(), ["echo"])

    def test_duplicate_registration_fails(self) -> None:
        factory = ProviderFactory()
        factory.register("echo", lambda: EchoProvider())

        with self.assertRaises(ProviderFactoryError):
            factory.register("echo", lambda: EchoProvider())


if __name__ == "__main__":
    unittest.main()
