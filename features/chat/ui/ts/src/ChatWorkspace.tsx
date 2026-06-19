import { useEffect, useRef } from "react";
import { ChatComposer } from "./ChatComposer";
import { ChatThread } from "./ChatThread";
import { ConversationList } from "./ConversationList";
import type { ChatAdapter, ChatLabels, ChatRequest } from "./types";
import { useChatSession } from "./useChatSession";

export function ChatWorkspace({
  adapter,
  labels = {},
  baseRequest,
  showConversations = true,
  className = "",
  nowLabel,
}: {
  adapter: ChatAdapter;
  labels?: ChatLabels;
  baseRequest?: Partial<ChatRequest>;
  showConversations?: boolean;
  className?: string;
  nowLabel?: () => string;
}) {
  const endRef = useRef<HTMLDivElement>(null);
  const session = useChatSession({ adapter, baseRequest, nowLabel });

  useEffect(() => {
    if (typeof endRef.current?.scrollIntoView === "function") {
      endRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [session.messages.length, session.streamDraft?.text]);

  return (
    <section className={`m-chat-workspace ${className}`.trim()}>
      <header className="m-chat-workspace__header">
        <div>
          {labels.title ? <h2>{labels.title}</h2> : null}
          {labels.lead ? <p>{labels.lead}</p> : null}
        </div>
        <button type="button" onClick={session.newChat}>
          {labels.newConversation ?? "New conversation"}
        </button>
      </header>
      {session.error ? (
        <p role="alert" className="m-chat-error">
          {session.error}
        </p>
      ) : null}
      <div className="m-chat-workspace__body">
        {showConversations ? (
          <ConversationList
            conversations={session.conversations}
            activeId={session.chatId}
            title={labels.conversations ?? "Conversations"}
            emptyMessage={labels.emptyConversations ?? "No conversations"}
            onSelect={(id) => void session.loadConversation(id)}
            onNew={session.newChat}
            newLabel={labels.newConversation ?? "New"}
          />
        ) : null}
        <main className="m-chat-workspace__main">
          <ChatThread
            messages={session.messages}
            streamDraft={session.streamDraft}
            loading={session.loadingHistory}
            loadingMessage={String(labels.loadingHistory ?? "Loading history...")}
            emptyMessage={String(labels.emptyThread ?? "No messages yet")}
            endRef={endRef}
          />
          <ChatComposer
            input={session.input}
            onInputChange={session.setInput}
            onSend={() => void session.send()}
            busy={session.loading}
            placeholder={labels.inputPlaceholder ?? "Write a message"}
            sendLabel={labels.send ?? "Send"}
            sendingLabel={labels.sending ?? "Sending..."}
            pendingConfirmations={session.pendingConfirmations}
            confirmPendingLabel={labels.confirmPending ?? "Confirm"}
            onConfirmPending={() => void session.confirmPending()}
          />
        </main>
      </div>
    </section>
  );
}
