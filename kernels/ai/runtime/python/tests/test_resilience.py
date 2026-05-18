from __future__ import annotations

import unittest

from runtime.resilience import CircuitBreaker, CircuitBreakerOpenError, CircuitBreakerState


class ResilienceTests(unittest.IsolatedAsyncioTestCase):
    async def test_circuit_breaker_opens_after_failures(self) -> None:
        breaker = CircuitBreaker(failure_threshold=2, recovery_timeout_seconds=60)

        async def failing() -> str:
            raise RuntimeError("boom")

        with self.assertRaises(RuntimeError):
            await breaker.call(failing)
        with self.assertRaises(RuntimeError):
            await breaker.call(failing)

        self.assertEqual(breaker.state, CircuitBreakerState.OPEN)
        with self.assertRaises(CircuitBreakerOpenError):
            await breaker.call(failing)


if __name__ == "__main__":
    unittest.main()
