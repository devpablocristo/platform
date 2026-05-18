from runtime.memory.operational import (
    build_operational_memory_view,
    capture_operational_turn,
    consolidate_operational_memory,
    ensure_operational_memory,
)


def test_capture_operational_turn_stores_generic_memory() -> None:
    container: dict[str, object] = {}

    capture_operational_turn(
        container,
        user_id="user-1",
        routed_agent="general",
        user_message="Necesito revisar ventas",
        assistant_reply="Voy a ayudarte con eso.",
        tool_calls=["get_recent_sales"],
        pending_confirmations=["create_sale"],
        confirmed_actions=set(),
        business_facts=["Vendemos al por mayor"],
        user_preferences=["El usuario prefiere respuestas breves."],
    )

    memory = ensure_operational_memory(container)
    assert memory["business_facts"][0]["text"] == "Vendemos al por mayor"
    assert memory["open_loops"][0]["actions"] == ["create_sale"]
    assert memory["user_profiles"]["user-1"]["active_preferences"] == ["El usuario prefiere respuestas breves."]


def test_consolidate_operational_memory_dedupes_decisions_and_preferences() -> None:
    container = {
        "memory": {
            "business_facts": [
                {"text": "Vendemos al por mayor", "kind": "explicit_business_fact", "times_seen": 2},
                {"text": "Vendemos al por mayor", "kind": "explicit_business_fact", "times_seen": 1},
            ],
            "stable_business_facts": [],
            "open_loops": [],
            "decisions": [
                {"action": "create_sale", "summary": "Confirmar venta urgente", "agent": "sales"},
                {"action": "create_sale", "summary": "Confirmar venta urgente", "agent": "sales"},
            ],
            "recent_threads": [],
            "user_profiles": {
                "user-1": {
                    "preferences": [
                        "El usuario prefiere respuestas breves.",
                        "El usuario prefiere respuestas breves.",
                    ],
                    "recent_topics": ["sales: revisar ventas", "sales: revisar ventas"],
                }
            },
        }
    }

    consolidate_operational_memory(container)
    memory = ensure_operational_memory(container)

    assert "Vendemos al por mayor" in memory["stable_business_facts"]
    assert len(memory["decisions"]) == 1
    assert memory["user_profiles"]["user-1"]["active_preferences"] == ["El usuario prefiere respuestas breves."]


def test_build_operational_memory_view_returns_curated_sections() -> None:
    container: dict[str, object] = {}
    capture_operational_turn(
        container,
        user_id="user-1",
        routed_agent="sales",
        user_message="Necesito revisar ventas",
        assistant_reply="Necesito confirmación para create_sale.",
        tool_calls=["get_recent_sales"],
        pending_confirmations=["create_sale"],
        confirmed_actions={"create_quote"},
        business_facts=["Vendemos al por mayor"],
        user_preferences=["El usuario prefiere respuestas con tablas cuando agregan valor."],
    )

    view = build_operational_memory_view(container, "user-1")

    assert "Vendemos al por mayor" in view["stable_business_facts"]
    assert any("create_sale" in item for item in view["open_loops"])
    assert any("create_quote" in item for item in view["decisions"])
    assert "El usuario prefiere respuestas con tablas cuando agregan valor." in view["active_preferences"]
