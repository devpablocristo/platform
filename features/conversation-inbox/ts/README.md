# @devpablocristo/platform-conversation-inbox

Componente React reutilizable para inbox de conversaciones (chat / notificaciones).
Wrappea `@devpablocristo/platform-notification-feed` con afordancias de
selección y slot de acciones por fila.

## Instalación

```bash
npm install @devpablocristo/platform-conversation-inbox
```

## Uso

```tsx
import {
  ConversationInbox,
  type ConversationInboxItem,
} from '@devpablocristo/platform-conversation-inbox'
import '@devpablocristo/platform-notification-feed/styles.css'
import '@devpablocristo/platform-conversation-inbox/styles.css'

const items: ConversationInboxItem[] = conversations.map(c => ({
  id: c.id,
  contactName: c.title,
  timestamp: relativeTime(c.created_at),
  unread: c.id !== selectedId,
  tone: c.id === selectedId ? 'attention' : 'default',
  actions: <button onClick={() => onOpen(c.id)}>Abrir</button>,
}))

<ConversationInbox items={items} loading={loading} emptyMessage="Sin conversaciones" />
```

## Peer deps

- `react`: `^18.0.0 || ^19.0.0`
- `react-dom`: `^18.0.0 || ^19.0.0`

## Consumidores

companion/console, pymes/frontend
