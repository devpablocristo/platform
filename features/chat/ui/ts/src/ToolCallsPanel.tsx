import type { ReactNode } from "react";
import { summarizeToolCall } from "./utils";

export function ToolCallsPanel({
  toolCalls = [],
  title = "Tools",
}: {
  toolCalls?: unknown[];
  title?: ReactNode;
}) {
  if (toolCalls.length === 0) return null;
  return (
    <section className="m-chat-tool-calls" aria-label={String(title)}>
      <strong>{title}</strong>
      <ul>
        {toolCalls.map((tool, index) => (
          <li key={`${summarizeToolCall(tool)}-${index}`}>{summarizeToolCall(tool)}</li>
        ))}
      </ul>
    </section>
  );
}
