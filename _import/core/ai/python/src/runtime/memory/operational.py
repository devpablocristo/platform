"""Reusable operational memory primitives for product assistants."""

from __future__ import annotations

from typing import Any


def normalize_memory_text(value: Any, *, limit: int = 240) -> str:
    text = " ".join(str(value or "").strip().split())
    return text[:limit]


def ensure_operational_memory(container: dict[str, Any]) -> dict[str, Any]:
    memory = container.setdefault("memory", {})
    memory.setdefault("business_facts", [])
    memory.setdefault("stable_business_facts", [])
    memory.setdefault("open_loops", [])
    memory.setdefault("decisions", [])
    memory.setdefault("recent_threads", [])
    memory.setdefault("user_profiles", {})
    return memory


def append_unique_list(items: list[Any], item: Any, *, limit: int = 10) -> None:
    if item in items:
        items.remove(item)
    items.append(item)
    if len(items) > limit:
        del items[:-limit]


def upsert_memory_fact(items: list[dict[str, Any]], text: str, kind: str, *, limit: int = 12) -> None:
    normalized = normalize_memory_text(text)
    if not normalized:
        return
    existing = next((entry for entry in items if normalize_memory_text(entry.get("text")) == normalized), None)
    if existing is not None:
        existing["times_seen"] = int(existing.get("times_seen", 1) or 1) + 1
        existing["kind"] = kind
    else:
        items.append({"text": normalized, "kind": kind, "times_seen": 1})
    if len(items) > limit:
        del items[:-limit]


def remove_resolved_open_loops(open_loops: list[dict[str, Any]], confirmed_actions: set[str]) -> None:
    if not confirmed_actions:
        return
    pending: list[dict[str, Any]] = []
    for loop in open_loops:
        actions = {str(action).strip().lower() for action in loop.get("actions", []) if str(action).strip()}
        if actions and actions.intersection(confirmed_actions):
            continue
        pending.append(loop)
    open_loops[:] = pending


def consolidate_operational_memory(container: dict[str, Any]) -> dict[str, Any]:
    memory = ensure_operational_memory(container)
    business_facts = list(memory.get("business_facts", []))
    business_facts.sort(
        key=lambda item: (
            int(item.get("times_seen", 1) or 1),
            normalize_memory_text(item.get("text")),
        ),
        reverse=True,
    )
    business_facts = business_facts[-12:]
    stable_facts = [
        normalize_memory_text(item.get("text"))
        for item in business_facts
        if normalize_memory_text(item.get("text")) and int(item.get("times_seen", 1) or 1) >= 1
    ][-6:]
    memory["business_facts"] = business_facts
    memory["stable_business_facts"] = stable_facts

    decisions = memory.get("decisions", [])
    deduped_decisions: list[dict[str, Any]] = []
    seen_decisions: set[tuple[str, str]] = set()
    for item in decisions:
        action = normalize_memory_text(item.get("action"))
        summary = normalize_memory_text(item.get("summary"))
        if not action and not summary:
            continue
        key = (action, summary)
        if key in seen_decisions:
            continue
        seen_decisions.add(key)
        deduped_decisions.append(
            {
                "action": action,
                "summary": summary,
                "agent": normalize_memory_text(item.get("agent")),
            }
        )
    memory["decisions"] = deduped_decisions[-8:]

    user_profiles = memory.get("user_profiles", {})
    for user_id, profile in list(user_profiles.items()):
        preferences = [normalize_memory_text(item) for item in profile.get("preferences", [])]
        preferences = [item for item in preferences if item]
        deduped_preferences: list[str] = []
        for item in preferences:
            if item not in deduped_preferences:
                deduped_preferences.append(item)
        recent_topics = [normalize_memory_text(item) for item in profile.get("recent_topics", [])]
        recent_topics = [item for item in recent_topics if item]
        deduped_topics: list[str] = []
        for item in recent_topics:
            if item not in deduped_topics:
                deduped_topics.append(item)
        user_profiles[user_id] = {
            "preferences": deduped_preferences[-8:],
            "active_preferences": deduped_preferences[-4:],
            "recent_topics": deduped_topics[-8:],
        }
    memory["user_profiles"] = user_profiles
    return container


def capture_operational_turn(
    container: dict[str, Any],
    *,
    user_id: str | None,
    routed_agent: str,
    user_message: str,
    assistant_reply: str,
    tool_calls: list[str] | None = None,
    pending_confirmations: list[str] | None = None,
    confirmed_actions: set[str] | None = None,
    business_facts: list[str] | None = None,
    user_preferences: list[str] | None = None,
) -> dict[str, Any]:
    memory = ensure_operational_memory(container)
    normalized_user_message = normalize_memory_text(user_message)
    normalized_reply = normalize_memory_text(assistant_reply)
    confirmed = {action.strip().lower() for action in (confirmed_actions or set()) if action.strip()}

    for fact in business_facts or []:
        upsert_memory_fact(memory["business_facts"], fact, "explicit_business_fact")

    if user_id:
        user_profiles = memory.setdefault("user_profiles", {})
        profile = user_profiles.setdefault(user_id, {"preferences": [], "recent_topics": []})
        for preference in user_preferences or []:
            normalized_preference = normalize_memory_text(preference)
            if normalized_preference:
                append_unique_list(profile["preferences"], normalized_preference, limit=8)
        topic = f"{routed_agent}: {normalized_user_message[:120]}".strip()
        if topic:
            append_unique_list(profile["recent_topics"], topic, limit=8)

    recent_threads = memory.setdefault("recent_threads", [])
    recent_threads.append(
        {
            "agent": routed_agent,
            "user_message": normalized_user_message[:160],
            "assistant_reply": normalized_reply[:160],
            "tool_calls": list(tool_calls or [])[:5],
        }
    )
    if len(recent_threads) > 10:
        del recent_threads[:-10]

    open_loops = memory.setdefault("open_loops", [])
    remove_resolved_open_loops(open_loops, confirmed)
    pending = [action.strip() for action in (pending_confirmations or []) if action.strip()]
    if pending:
        open_loops.append(
            {
                "summary": normalized_user_message[:160],
                "actions": pending,
                "agent": routed_agent,
            }
        )
        if len(open_loops) > 8:
            del open_loops[:-8]

    if confirmed:
        decisions = memory.setdefault("decisions", [])
        for action in sorted(confirmed):
            decisions.append(
                {
                    "action": action,
                    "summary": normalized_user_message[:160] or normalized_reply[:160],
                    "agent": routed_agent,
                }
            )
        if len(decisions) > 10:
            del decisions[:-10]

    consolidate_operational_memory(container)
    return container


def build_operational_memory_view(container: dict[str, Any], user_id: str | None = None) -> dict[str, list[str]]:
    consolidate_operational_memory(container)
    memory = ensure_operational_memory(container)
    view: dict[str, list[str]] = {
        "stable_business_facts": [
            normalize_memory_text(item) for item in memory.get("stable_business_facts", []) if normalize_memory_text(item)
        ],
        "open_loops": [],
        "decisions": [],
        "active_preferences": [],
        "recent_topics": [],
    }
    for loop in memory.get("open_loops", [])[-3:]:
        summary = normalize_memory_text(loop.get("summary"))
        actions = ", ".join(str(action).strip() for action in loop.get("actions", []) if str(action).strip())
        detail = f"{summary}. Pendiente: {actions}." if actions else summary
        if detail:
            view["open_loops"].append(detail)
    for decision in memory.get("decisions", [])[-3:]:
        action = normalize_memory_text(decision.get("action"))
        summary = normalize_memory_text(decision.get("summary"))
        detail = f"{action}: {summary}" if action and summary else action or summary
        if detail:
            view["decisions"].append(detail)
    if user_id:
        profile = memory.get("user_profiles", {}).get(user_id, {})
        view["active_preferences"] = [
            normalize_memory_text(item) for item in profile.get("active_preferences", []) if normalize_memory_text(item)
        ]
        view["recent_topics"] = [
            normalize_memory_text(item) for item in profile.get("recent_topics", [])[-3:] if normalize_memory_text(item)
        ]
    return view
