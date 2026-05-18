"""Modelos Pydantic genéricos para chat AI — agnósticos de producto."""

from __future__ import annotations

from typing import Annotated, Literal

from pydantic import AliasChoices, BaseModel, Field

from runtime.domain.contracts import DEFAULT_LANGUAGE_CODE, OUTPUT_KIND_CHAT_REPLY


class ChatRequest(BaseModel):
    """Request genérico de chat. Los productos extienden con campos propios."""

    chat_id: str | None = Field(
        default=None,
        validation_alias=AliasChoices("chat_id", "conversation_id"),
        serialization_alias="chat_id",
    )
    message: str = Field(min_length=1, max_length=4000)
    preferred_language: str | None = Field(
        default=None,
        description="ISO language code for AI-generated content (e.g. 'es', 'en').",
    )


class ChatAction(BaseModel):
    """Acción interactiva que el frontend puede renderizar como botón."""

    id: str = Field(min_length=1)
    label: str = Field(min_length=1)
    kind: Literal["send_message", "open_url", "confirm_action"]
    style: Literal["primary", "secondary", "ghost"] = "secondary"
    message: str | None = None
    url: str | None = None


class ChatTextBlock(BaseModel):
    type: Literal["text"]
    text: str = Field(min_length=1)


class ChatActionsBlock(BaseModel):
    type: Literal["actions"]
    actions: list[ChatAction] = Field(default_factory=list)


class InsightCardHighlight(BaseModel):
    label: str = Field(min_length=1)
    value: str = Field(min_length=1)


class ChatInsightCardBlock(BaseModel):
    type: Literal["insight_card"]
    title: str = Field(min_length=1)
    summary: str = Field(min_length=1)
    scope: str | None = None
    highlights: list[InsightCardHighlight] = Field(default_factory=list)
    recommendations: list[str] = Field(default_factory=list)


class ChatKpiItem(BaseModel):
    label: str = Field(min_length=1)
    value: str = Field(min_length=1)
    trend: Literal["up", "down", "flat", "unknown"] | None = None
    context: str | None = None


class ChatKpiGroupBlock(BaseModel):
    type: Literal["kpi_group"]
    title: str | None = None
    items: list[ChatKpiItem] = Field(default_factory=list)


class ChatTableBlock(BaseModel):
    type: Literal["table"]
    title: str = Field(min_length=1)
    columns: list[str] = Field(default_factory=list)
    rows: list[list[str]] = Field(default_factory=list)
    empty_state: str | None = None


ChatBlock = Annotated[
    ChatTextBlock | ChatActionsBlock | ChatInsightCardBlock | ChatKpiGroupBlock | ChatTableBlock,
    Field(discriminator="type"),
]


class ChatResponse(BaseModel):
    """Response genérico de chat. Los productos extienden con campos propios."""

    request_id: str
    output_kind: str = Field(default=OUTPUT_KIND_CHAT_REPLY)
    content_language: str = Field(default=DEFAULT_LANGUAGE_CODE)
    chat_id: str = Field(serialization_alias="chat_id")
    reply: str
    tokens_used: int
    tool_calls: list[str] = Field(default_factory=list)
    blocks: list[ChatBlock] = Field(default_factory=list)
