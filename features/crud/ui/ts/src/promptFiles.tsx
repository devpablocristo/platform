import type { CSSProperties, ReactNode } from "react";
import { useRef } from "react";

export type ReadonlyMetadataItem = {
  label: string;
  value: ReactNode;
};

export type TextFileUploadResult<TContext = unknown> = {
  file: File;
  fileName: string;
  content: string;
  context?: TContext;
};

export type UseTextFileUploadOptions<TContext = unknown> = {
  accept?: string;
  extensions?: string[];
  allowEmpty?: boolean;
  emptyFileMessage?: string;
  invalidFileMessage?: string;
  onLoad: (result: TextFileUploadResult<TContext>) => void | Promise<void>;
  onError?: (error: Error) => void;
};

export type UseTextFileUploadResult<TContext = unknown> = {
  inputProps: TextFileInputProps;
  inputRef: (node: HTMLInputElement | null) => void;
  open: (context?: TContext) => void;
};

export type TextFileInputProps = {
  accept: string;
  onChange: (event: { currentTarget: HTMLInputElement }) => void;
  style: CSSProperties;
  type: "file";
};

export type ReadonlyContentViewerProps = {
  title: ReactNode;
  ariaLabel?: string;
  subtitle?: ReactNode;
  metadata?: ReadonlyMetadataItem[];
  content?: string;
  emptyContent?: ReactNode;
  closeLabel?: string;
  onClose?: () => void;
  className?: string;
  contentClassName?: string;
};

export type FileUploadReviewProps = {
  title: ReactNode;
  ariaLabel?: string;
  subtitle?: ReactNode;
  metadata?: ReadonlyMetadataItem[];
  content: string;
  cancelLabel?: string;
  confirmLabel?: string;
  onCancel: () => void;
  onConfirm: () => void;
  className?: string;
  contentClassName?: string;
};

export type PromptEditorReviewProps = {
  title: ReactNode;
  ariaLabel?: string;
  subtitle?: ReactNode;
  metadata?: ReadonlyMetadataItem[];
  content: string;
  onContentChange?: (value: string) => void;
  readOnly?: boolean;
  cancelLabel?: string;
  confirmLabel?: string;
  onCancel: () => void;
  onConfirm: () => void;
  className?: string;
  editorClassName?: string;
};

export type DownloadZipFileEntry = {
  fileName: string;
  content: string;
};

export function useTextFileUpload<TContext = unknown>(
  options: UseTextFileUploadOptions<TContext>,
): UseTextFileUploadResult<TContext> {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const contextRef = useRef<TContext | undefined>(undefined);
  const extensions = options.extensions ?? [".md"];
  const accept = options.accept ?? extensions.join(",");

  return {
    inputProps: {
      type: "file",
      accept,
      style: { display: "none" },
      onChange: (event) => {
        void handleTextFileInputChange(event, contextRef.current, options, extensions);
        contextRef.current = undefined;
      },
    },
    inputRef: (node) => {
      inputRef.current = node;
    },
    open: (context?: TContext) => {
      contextRef.current = context;
      inputRef.current?.click();
    },
  };
}

export function ReadonlyContentViewer({
  title,
  ariaLabel,
  subtitle,
  metadata = [],
  content,
  emptyContent,
  closeLabel = "Close",
  onClose,
  className,
  contentClassName,
}: ReadonlyContentViewerProps) {
  return (
    <section aria-label={ariaLabel ?? stringLabel(title)} className={className} style={cardStyle}>
      <Header title={title} subtitle={subtitle}>
        {onClose ? (
          <button type="button" style={buttonStyle} onClick={onClose}>
            {closeLabel}
          </button>
        ) : null}
      </Header>
      <Metadata items={metadata} />
      {content ? (
        <pre className={contentClassName} style={contentStyle}>
          {content}
        </pre>
      ) : (
        <p style={mutedStyle}>{emptyContent}</p>
      )}
    </section>
  );
}

export function FileUploadReview({
  title,
  ariaLabel,
  subtitle,
  metadata = [],
  content,
  cancelLabel = "Cancel",
  confirmLabel = "Confirm",
  onCancel,
  onConfirm,
  className,
  contentClassName,
}: FileUploadReviewProps) {
  return (
    <section aria-label={ariaLabel ?? stringLabel(title)} className={className} style={cardStyle}>
      <Header title={title} subtitle={subtitle}>
        <div style={actionsStyle}>
          <button type="button" style={buttonStyle} onClick={onCancel}>
            {cancelLabel}
          </button>
          <button type="button" style={primaryButtonStyle} onClick={onConfirm}>
            {confirmLabel}
          </button>
        </div>
      </Header>
      <Metadata items={metadata} />
      <pre className={contentClassName} style={contentStyle}>
        {content}
      </pre>
    </section>
  );
}

export function PromptEditorReview({
  title,
  ariaLabel,
  subtitle,
  metadata = [],
  content,
  onContentChange,
  readOnly = false,
  cancelLabel = "Cancel",
  confirmLabel = "Confirm",
  onCancel,
  onConfirm,
  className,
  editorClassName,
}: PromptEditorReviewProps) {
  return (
    <section aria-label={ariaLabel ?? stringLabel(title)} className={className} style={cardStyle}>
      <Header title={title} subtitle={subtitle}>
        <div style={actionsStyle}>
          <button type="button" style={buttonStyle} onClick={onCancel}>
            {cancelLabel}
          </button>
          <button type="button" style={primaryButtonStyle} onClick={onConfirm}>
            {confirmLabel}
          </button>
        </div>
      </Header>
      <Metadata items={metadata} />
      <textarea
        aria-label={ariaLabel ? `${ariaLabel} content` : "Prompt content"}
        className={editorClassName}
        value={content}
        readOnly={readOnly}
        onChange={(event) => onContentChange?.(event.currentTarget.value)}
        style={editorStyle}
      />
    </section>
  );
}

export function downloadTextFile(fileName: string, content: string, type = "text/markdown;charset=utf-8"): void {
  downloadBlob(fileName, new Blob([ensureTrailingNewline(content)], { type }));
}

export function downloadZipFile(fileName: string, files: DownloadZipFileEntry[]): void {
  const entries = files.filter((file) => file.content.trim());
  if (entries.length === 0) {
    throw new Error("No files to download");
  }
  downloadBlob(fileName, new Blob([createZip(entries)], { type: "application/zip" }));
}

export function ensureTrailingNewline(value: string): string {
  return value.endsWith("\n") ? value : `${value}\n`;
}

export function safeFileName(value: string, fallback = "file"): string {
  const parts = value.split(".");
  const extension = parts.length > 1 ? `.${parts.pop()}` : "";
  const base = parts.join(".") || value;
  const safeBase = base
    .toLowerCase()
    .replace(/[^a-z0-9_-]+/g, "_")
    .replace(/_+/g, "_")
    .replace(/^_+|_+$/g, "");
  const safeExtension = extension.toLowerCase().replace(/[^a-z0-9.]+/g, "");
  return `${safeBase || fallback}${safeExtension}`;
}

async function handleTextFileInputChange<TContext>(
  event: { currentTarget: HTMLInputElement },
  context: TContext | undefined,
  options: UseTextFileUploadOptions<TContext>,
  extensions: string[],
): Promise<void> {
  const input = event.currentTarget;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;
  try {
    if (!hasAllowedExtension(file.name, extensions)) {
      throw new Error(options.invalidFileMessage ?? "Invalid file type");
    }
    const content = await file.text();
    if (!options.allowEmpty && !content.trim()) {
      throw new Error(options.emptyFileMessage ?? "File is empty");
    }
    await options.onLoad({ file, fileName: file.name, content, context });
  } catch (error) {
    options.onError?.(error instanceof Error ? error : new Error("File upload failed"));
  }
}

function hasAllowedExtension(fileName: string, extensions: string[]): boolean {
  const lower = fileName.toLowerCase();
  return extensions.some((extension) => lower.endsWith(extension.toLowerCase()));
}

function downloadBlob(fileName: string, blob: Blob): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function createZip(files: DownloadZipFileEntry[]): ArrayBuffer {
  const encoder = new TextEncoder();
  const localParts: Uint8Array[] = [];
  const centralParts: Uint8Array[] = [];
  let offset = 0;

  for (const file of files) {
    const name = encoder.encode(safeFileName(file.fileName));
    const data = encoder.encode(ensureTrailingNewline(file.content));
    const crc = crc32(data);
    const local = concatBytes([
      u32(0x04034b50),
      u16(20),
      u16(0),
      u16(0),
      u16(0),
      u16(0),
      u32(crc),
      u32(data.length),
      u32(data.length),
      u16(name.length),
      u16(0),
      name,
      data,
    ]);
    const central = concatBytes([
      u32(0x02014b50),
      u16(20),
      u16(20),
      u16(0),
      u16(0),
      u16(0),
      u16(0),
      u32(crc),
      u32(data.length),
      u32(data.length),
      u16(name.length),
      u16(0),
      u16(0),
      u16(0),
      u16(0),
      u32(0),
      u32(offset),
      name,
    ]);
    localParts.push(local);
    centralParts.push(central);
    offset += local.length;
  }

  const central = concatBytes(centralParts);
  const end = concatBytes([
    u32(0x06054b50),
    u16(0),
    u16(0),
    u16(files.length),
    u16(files.length),
    u32(central.length),
    u32(offset),
    u16(0),
  ]);
  const bytes = concatBytes([...localParts, central, end]);
  return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer;
}

function concatBytes(parts: Uint8Array[]): Uint8Array {
  const total = parts.reduce((sum, part) => sum + part.length, 0);
  const out = new Uint8Array(total);
  let offset = 0;
  for (const part of parts) {
    out.set(part, offset);
    offset += part.length;
  }
  return out;
}

function u16(value: number): Uint8Array {
  const out = new Uint8Array(2);
  const view = new DataView(out.buffer);
  view.setUint16(0, value, true);
  return out;
}

function u32(value: number): Uint8Array {
  const out = new Uint8Array(4);
  const view = new DataView(out.buffer);
  view.setUint32(0, value >>> 0, true);
  return out;
}

function crc32(data: Uint8Array): number {
  let crc = 0xffffffff;
  for (const byte of data) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit += 1) {
      crc = (crc >>> 1) ^ (0xedb88320 & -(crc & 1));
    }
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function Header({ title, subtitle, children }: { title: ReactNode; subtitle?: ReactNode; children?: ReactNode }) {
  return (
    <div style={headerStyle}>
      <div>
        <h2 style={titleStyle}>{title}</h2>
        {subtitle ? <p style={subtitleStyle}>{subtitle}</p> : null}
      </div>
      {children}
    </div>
  );
}

function stringLabel(value: ReactNode): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function Metadata({ items }: { items: ReadonlyMetadataItem[] }) {
  if (items.length === 0) return null;
  return (
    <dl style={metadataStyle}>
      {items.map((item) => (
        <div key={item.label}>
          <dt style={metadataLabelStyle}>{item.label}</dt>
          <dd style={metadataValueStyle}>{item.value}</dd>
        </div>
      ))}
    </dl>
  );
}

const cardStyle = {
  marginBottom: "1rem",
  padding: "1rem",
  border: "1px solid var(--crud-border, #d8dee6)",
  borderRadius: 8,
  background: "var(--crud-card-bg, #fff)",
} satisfies CSSProperties;

const headerStyle = {
  display: "flex",
  alignItems: "flex-start",
  justifyContent: "space-between",
  gap: "1rem",
} satisfies CSSProperties;

const titleStyle = { margin: 0, fontSize: "1.25rem" } satisfies CSSProperties;
const subtitleStyle = { margin: "0.35rem 0 0", color: "var(--crud-muted, #667383)" } satisfies CSSProperties;
const mutedStyle = { margin: "1rem 0 0", color: "var(--crud-muted, #667383)" } satisfies CSSProperties;
const actionsStyle = { display: "flex", gap: "0.5rem", flexWrap: "wrap", justifyContent: "flex-end" } satisfies CSSProperties;
const buttonStyle = {
  minHeight: 34,
  padding: "0 0.75rem",
  border: "1px solid var(--crud-border, #cfd6dd)",
  borderRadius: 8,
  background: "var(--crud-button-bg, #fff)",
  color: "var(--crud-text, #17202a)",
  cursor: "pointer",
} satisfies CSSProperties;
const primaryButtonStyle = {
  ...buttonStyle,
  borderColor: "var(--crud-primary, #18b99a)",
  background: "var(--crud-primary, #18b99a)",
  color: "var(--crud-primary-text, #fff)",
} satisfies CSSProperties;
const metadataStyle = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(11rem, 1fr))",
  gap: "0.75rem",
  margin: "1rem 0 0",
} satisfies CSSProperties;
const metadataLabelStyle = { color: "var(--crud-muted, #667383)", fontSize: "0.85rem" } satisfies CSSProperties;
const metadataValueStyle = { margin: "0.15rem 0 0" } satisfies CSSProperties;
const contentStyle = {
  margin: "1rem 0 0",
  maxHeight: "32rem",
  overflow: "auto",
  whiteSpace: "pre-wrap",
  borderRadius: 6,
  padding: "0.875rem",
  background: "var(--crud-subtle, #f6f7f9)",
} satisfies CSSProperties;

const editorStyle = {
  margin: "1rem 0 0",
  width: "100%",
  minHeight: "22rem",
  overflow: "auto",
  whiteSpace: "pre-wrap",
  border: "1px solid var(--crud-border, #cfd6dd)",
  borderRadius: 6,
  padding: "0.875rem",
  background: "var(--crud-subtle, #f6f7f9)",
  color: "var(--crud-text, #17202a)",
  fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
  fontSize: "0.9rem",
  lineHeight: 1.5,
  boxSizing: "border-box",
} satisfies CSSProperties;
