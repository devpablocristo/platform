from __future__ import annotations

import logging
import uuid
from dataclasses import dataclass
from typing import Any

import httpx

logger = logging.getLogger(__name__)

DEFAULT_TIMEOUT_SECONDS = 10.0
DEFAULT_FALLBACK_DECISION = "require_approval"


@dataclass(frozen=True)
class ReviewRequester:
    requester_type: str = "service"
    requester_id: str = "ai-service"
    requester_name: str = "AI Service"


@dataclass(frozen=True)
class SubmitResponse:
    request_id: str
    decision: str
    risk_level: str
    decision_reason: str
    status: str
    approval_id: str | None = None
    approval_expires_at: str | None = None


@dataclass(frozen=True)
class RequestStatus:
    id: str
    decision: str
    status: str
    decided_by: str | None = None
    decision_note: str | None = None


@dataclass(frozen=True)
class PolicyInfo:
    id: str
    name: str
    action_type: str
    expression: str
    effect: str
    mode: str
    created_at: str = ""
    updated_at: str = ""


@dataclass(frozen=True)
class ActionTypeInfo:
    name: str
    risk_class: str
    enabled: bool = True


@dataclass(frozen=True)
class ApprovalInfo:
    id: str
    request_id: str
    status: str
    action_type: str = ""
    target_resource: str = ""
    reason: str = ""
    risk_level: str = ""
    ai_summary: str | None = None
    created_at: str = ""
    expires_at: str | None = None


class ReviewClient:
    """Async client for review and governance APIs."""

    def __init__(
        self,
        base_url: str,
        api_key: str,
        *,
        requester: ReviewRequester | None = None,
        timeout_seconds: float = DEFAULT_TIMEOUT_SECONDS,
        client: httpx.AsyncClient | None = None,
    ) -> None:
        self._base_url = base_url.rstrip("/")
        self._api_key = api_key
        self._requester = requester or ReviewRequester()
        self._http = client or httpx.AsyncClient(
            base_url=self._base_url,
            timeout=timeout_seconds,
            headers={"X-API-Key": self._api_key, "Content-Type": "application/json"},
        )

    async def close(self) -> None:
        await self._http.aclose()

    async def submit_request(
        self,
        *,
        action_type: str,
        target_system: str = "",
        target_resource: str = "",
        params: dict[str, Any] | None = None,
        reason: str = "",
        context: str = "",
    ) -> SubmitResponse:
        idempotency_key = str(uuid.uuid4())
        body = {
            "requester_type": self._requester.requester_type,
            "requester_id": self._requester.requester_id,
            "requester_name": self._requester.requester_name,
            "action_type": action_type,
            "target_system": target_system,
            "target_resource": target_resource,
            "params": params or {},
            "reason": reason,
            "context": context,
        }
        try:
            response = await self._http.post(
                "/v1/requests",
                json=body,
                headers={"Idempotency-Key": idempotency_key},
            )
            response.raise_for_status()
            data = response.json()
            approval = data.get("approval") or {}
            return SubmitResponse(
                request_id=str(data.get("request_id", "")),
                decision=str(data.get("decision", "")),
                risk_level=str(data.get("risk_level", "")),
                decision_reason=str(data.get("decision_reason", "")),
                status=str(data.get("status", "")),
                approval_id=str(approval.get("id", "")) or None,
                approval_expires_at=str(approval.get("expires_at", "")) or None,
            )
        except Exception:
            logger.exception("review_submit_failed", extra={"action_type": action_type})
            return SubmitResponse(
                request_id="",
                decision=DEFAULT_FALLBACK_DECISION,
                risk_level="unknown",
                decision_reason="Review service unavailable; fallback to require_approval",
                status="fallback",
            )

    async def get_request(self, request_id: str) -> RequestStatus:
        try:
            response = await self._http.get(f"/v1/requests/{request_id}")
            response.raise_for_status()
            data = response.json()
            return RequestStatus(
                id=str(data.get("id", "")),
                decision=str(data.get("decision", "")),
                status=str(data.get("status", "")),
                decided_by=data.get("decided_by"),
                decision_note=data.get("decision_note"),
            )
        except Exception:
            logger.exception("review_get_request_failed", extra={"request_id": request_id})
            return RequestStatus(id=request_id, decision="", status="unknown")

    async def report_result(
        self,
        request_id: str,
        *,
        success: bool,
        duration_ms: int = 0,
        details: str = "",
    ) -> None:
        try:
            response = await self._http.post(
                f"/v1/requests/{request_id}/result",
                json={"success": success, "duration_ms": duration_ms, "details": details},
            )
            response.raise_for_status()
        except Exception:
            logger.warning("review_report_result_failed", extra={"request_id": request_id})

    async def list_policies(self) -> list[PolicyInfo]:
        try:
            response = await self._http.get("/v1/policies")
            response.raise_for_status()
            payload = response.json()
            items = payload if isinstance(payload, list) else payload.get("policies", [])
            return [
                PolicyInfo(
                    id=str(policy.get("id", "")),
                    name=str(policy.get("name", "")),
                    action_type=str(policy.get("action_type", "")),
                    expression=str(policy.get("expression", "")),
                    effect=str(policy.get("effect", "")),
                    mode=str(policy.get("mode", "enforced")),
                    created_at=str(policy.get("created_at", "")),
                    updated_at=str(policy.get("updated_at", "")),
                )
                for policy in items
            ]
        except Exception:
            logger.exception("review_list_policies_failed")
            return []

    async def create_policy(
        self,
        *,
        name: str,
        action_type: str,
        expression: str,
        effect: str,
        mode: str = "enforced",
    ) -> PolicyInfo | None:
        try:
            response = await self._http.post(
                "/v1/policies",
                json={
                    "name": name,
                    "action_type": action_type,
                    "expression": expression,
                    "effect": effect,
                    "mode": mode,
                },
            )
            response.raise_for_status()
            policy = response.json()
            return PolicyInfo(
                id=str(policy.get("id", "")),
                name=str(policy.get("name", "")),
                action_type=str(policy.get("action_type", "")),
                expression=str(policy.get("expression", "")),
                effect=str(policy.get("effect", "")),
                mode=str(policy.get("mode", "enforced")),
                created_at=str(policy.get("created_at", "")),
                updated_at=str(policy.get("updated_at", "")),
            )
        except Exception:
            logger.exception("review_create_policy_failed")
            return None

    async def update_policy(self, policy_id: str, **kwargs: Any) -> PolicyInfo | None:
        try:
            response = await self._http.patch(f"/v1/policies/{policy_id}", json=kwargs)
            response.raise_for_status()
            policy = response.json()
            return PolicyInfo(
                id=str(policy.get("id", "")),
                name=str(policy.get("name", "")),
                action_type=str(policy.get("action_type", "")),
                expression=str(policy.get("expression", "")),
                effect=str(policy.get("effect", "")),
                mode=str(policy.get("mode", "enforced")),
            )
        except Exception:
            logger.exception("review_update_policy_failed")
            return None

    async def delete_policy(self, policy_id: str) -> bool:
        try:
            response = await self._http.delete(f"/v1/policies/{policy_id}")
            return response.status_code in (200, 204)
        except Exception:
            logger.exception("review_delete_policy_failed")
            return False

    async def list_action_types(self) -> list[ActionTypeInfo]:
        try:
            response = await self._http.get("/v1/action-types")
            response.raise_for_status()
            payload = response.json()
            items = payload if isinstance(payload, list) else payload.get("action_types", [])
            return [
                ActionTypeInfo(
                    name=str(item.get("name", "")),
                    risk_class=str(item.get("risk_class", "low")),
                    enabled=bool(item.get("enabled", True)),
                )
                for item in items
            ]
        except Exception:
            logger.exception("review_list_action_types_failed")
            return []

    async def list_pending_approvals(self) -> list[ApprovalInfo]:
        try:
            response = await self._http.get("/v1/approvals/pending")
            response.raise_for_status()
            payload = response.json()
            items = payload if isinstance(payload, list) else payload.get("approvals", [])
            return [
                ApprovalInfo(
                    id=str(item.get("id", "")),
                    request_id=str(item.get("request_id", "")),
                    status=str(item.get("status", "")),
                    action_type=str(item.get("action_type", "")),
                    target_resource=str(item.get("target_resource", "")),
                    reason=str(item.get("reason", "")),
                    risk_level=str(item.get("risk_level", "")),
                    ai_summary=item.get("ai_summary"),
                    created_at=str(item.get("created_at", "")),
                    expires_at=item.get("expires_at"),
                )
                for item in items
            ]
        except Exception:
            logger.exception("review_list_pending_approvals_failed")
            return []

    async def approve(self, approval_id: str, decided_by: str, note: str = "") -> bool:
        try:
            response = await self._http.post(
                f"/v1/approvals/{approval_id}/approve",
                json={"decided_by": decided_by, "note": note},
            )
            return response.status_code in (200, 204)
        except Exception:
            logger.exception("review_approve_failed")
            return False

    async def reject(self, approval_id: str, decided_by: str, note: str = "") -> bool:
        try:
            response = await self._http.post(
                f"/v1/approvals/{approval_id}/reject",
                json={"decided_by": decided_by, "note": note},
            )
            return response.status_code in (200, 204)
        except Exception:
            logger.exception("review_reject_failed")
            return False
