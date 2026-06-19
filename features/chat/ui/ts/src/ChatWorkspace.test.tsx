// @vitest-environment jsdom

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ChatWorkspace } from "./ChatWorkspace";
import type { ChatAdapter } from "./types";

describe("ChatWorkspace", () => {
  it("sends a normal message and renders the assistant reply", async () => {
    const adapter: ChatAdapter = {
      sendMessage: vi.fn(async () => ({ chatId: "chat-1", reply: "Hola" })),
      listConversations: vi.fn(async () => ({ items: [] })),
    };

    render(<ChatWorkspace adapter={adapter} labels={{ inputPlaceholder: "Mensaje" }} />);

    fireEvent.change(screen.getByLabelText("Mensaje"), { target: { value: "hola" } });
    fireEvent.click(screen.getByRole("button", { name: "Send" }));

    await waitFor(() => expect(screen.getByText("Hola")).toBeTruthy());
    expect(adapter.sendMessage).toHaveBeenCalledWith(expect.objectContaining({ message: "hola" }));
  });

  it("loads conversations from the adapter", async () => {
    const adapter: ChatAdapter = {
      sendMessage: vi.fn(),
      listConversations: vi.fn(async () => ({
        items: [{ id: "chat-1", title: "Consulta" }],
      })),
      getConversation: vi.fn(async () => ({
        id: "chat-1",
        messages: [{ role: "assistant", content: "Historial" }],
      })),
    };

    render(<ChatWorkspace adapter={adapter} />);

    await waitFor(() => expect(screen.getByText("Consulta")).toBeTruthy());
    fireEvent.click(screen.getByRole("button", { name: /Consulta/ }));
    await waitFor(() => expect(screen.getByText("Historial")).toBeTruthy());
  });
});
