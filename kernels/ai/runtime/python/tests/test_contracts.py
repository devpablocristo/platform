from runtime import (
    AIRequestContext,
    NotificationChatHandoff,
    OUTPUT_KIND_COPILOT_EXPLANATION,
    OUTPUT_KIND_INSIGHT_SUMMARY,
    ROUTING_SOURCE_COPILOT_AGENT,
    ROUTING_SOURCE_ORCHESTRATOR,
    SERVICE_KIND_INSIGHT,
    is_known_routing_source,
    normalize_notification_chat_handoff,
    normalize_routing_source,
)


def test_normalize_routing_source_keeps_known_values() -> None:
    assert normalize_routing_source(ROUTING_SOURCE_ORCHESTRATOR) == ROUTING_SOURCE_ORCHESTRATOR
    assert is_known_routing_source(ROUTING_SOURCE_ORCHESTRATOR) is True


def test_normalize_routing_source_falls_back_to_orchestrator() -> None:
    assert normalize_routing_source("unknown") == ROUTING_SOURCE_ORCHESTRATOR
    assert normalize_routing_source(None) == ROUTING_SOURCE_ORCHESTRATOR


def test_ai_request_context_keeps_shared_surface_metadata() -> None:
    agent_context = AIRequestContext(
        request_id="req-123",
        routed_agent="copilot",
        routing_source=ROUTING_SOURCE_COPILOT_AGENT,
        output_kind=OUTPUT_KIND_COPILOT_EXPLANATION,
    )
    service_context = AIRequestContext(
        request_id="req-456",
        service_kind=SERVICE_KIND_INSIGHT,
        output_kind=OUTPUT_KIND_INSIGHT_SUMMARY,
    )

    assert agent_context.request_id == "req-123"
    assert agent_context.routed_agent == "copilot"
    assert agent_context.routing_source == ROUTING_SOURCE_COPILOT_AGENT
    assert agent_context.output_kind == OUTPUT_KIND_COPILOT_EXPLANATION
    assert service_context.request_id == "req-456"
    assert service_context.service_kind == SERVICE_KIND_INSIGHT
    assert service_context.output_kind == OUTPUT_KIND_INSIGHT_SUMMARY


def test_normalize_notification_chat_handoff_keeps_canonical_fields() -> None:
    handoff = normalize_notification_chat_handoff(
        {
            "notification_id": "notif-1",
            "title": "Costo alto",
            "body": "Se detecto un desvio",
            "scope": "cost_overrun",
            "routed_agent": "copilot",
            "content_language": "ES",
            "suggested_user_message": "Explicame este insight",
            "kind": "insight",
            "entity_type": "insight",
            "entity_id": "ins-9",
        }
    )

    assert isinstance(handoff, NotificationChatHandoff)
    assert handoff.notification_id == "notif-1"
    assert handoff.routed_agent == "copilot"
    assert handoff.content_language == "es"
    assert handoff.source_notification_kind == "insight"
    assert handoff.entity_id == "ins-9"
