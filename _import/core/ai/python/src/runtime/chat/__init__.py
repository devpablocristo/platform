"""Primitivas reutilizables de chat para productos AI."""

from .contracts import (
    ChatAction,
    ChatActionsBlock,
    ChatBlock,
    ChatInsightCardBlock,
    ChatKpiGroupBlock,
    ChatKpiItem,
    ChatRequest,
    ChatResponse,
    ChatTableBlock,
    ChatTextBlock,
    InsightCardHighlight,
)
from .stream import StreamChatResult, stream_orchestrated_chat
from .blocks import (
    build_actions_block,
    build_insight_card_block,
    build_kpi_group_block,
    build_table_block,
    build_text_block,
)

__all__ = [
    "ChatAction",
    "ChatActionsBlock",
    "ChatBlock",
    "ChatInsightCardBlock",
    "ChatKpiGroupBlock",
    "ChatKpiItem",
    "ChatRequest",
    "ChatResponse",
    "ChatTableBlock",
    "ChatTextBlock",
    "InsightCardHighlight",
    "StreamChatResult",
    "build_actions_block",
    "build_insight_card_block",
    "build_kpi_group_block",
    "build_table_block",
    "build_text_block",
    "stream_orchestrated_chat",
]
