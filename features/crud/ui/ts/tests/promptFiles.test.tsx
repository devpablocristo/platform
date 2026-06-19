// @vitest-environment jsdom

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  FileUploadReview,
  PromptEditorReview,
  ReadonlyContentViewer,
  downloadTextFile,
  downloadZipFile,
  ensureTrailingNewline,
  safeFileName,
  useTextFileUpload,
} from "../src/promptFiles";

describe("prompt file helpers", () => {
  beforeEach(() => {
    vi.stubGlobal("URL", {
      createObjectURL: vi.fn(() => "blob:crud"),
      revokeObjectURL: vi.fn(),
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders readonly content", () => {
    render(
      <ReadonlyContentViewer
        title="Prompt"
        subtitle="clinical_summary"
        metadata={[{ label: "Version", value: "v1" }]}
        content="Use evidence."
      />,
    );

    expect(screen.getByText("Prompt")).toBeTruthy();
    expect(screen.getByText("clinical_summary")).toBeTruthy();
    expect(screen.getByText("Version")).toBeTruthy();
    expect(screen.getByText("Use evidence.")).toBeTruthy();
  });

  it("renders upload review and confirms", () => {
    const onConfirm = vi.fn();
    render(
      <FileUploadReview
        title="Review"
        metadata={[{ label: "File", value: "prompt.md" }]}
        content="New prompt"
        onCancel={vi.fn()}
        onConfirm={onConfirm}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("renders editable prompt review", () => {
    const onChange = vi.fn();
    const onConfirm = vi.fn();
    render(
      <PromptEditorReview
        title="Review"
        content="Old prompt"
        onContentChange={onChange}
        onCancel={vi.fn()}
        onConfirm={onConfirm}
      />,
    );

    fireEvent.change(screen.getByLabelText("Prompt content"), { target: { value: "New prompt" } });
    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));

    expect(onChange).toHaveBeenCalledWith("New prompt");
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("loads a markdown file and rejects invalid files", async () => {
    const onLoad = vi.fn();
    const onError = vi.fn();
    function TestUpload() {
      const upload = useTextFileUpload({
        onLoad,
        onError,
        invalidFileMessage: "invalid",
        emptyFileMessage: "empty",
      });
      return <input aria-label="upload" ref={upload.inputRef} {...upload.inputProps} />;
    }
    render(<TestUpload />);

    const input = screen.getByLabelText("upload") as HTMLInputElement;
    const md = new File(["hello"], "prompt.md", { type: "text/markdown" });
    fireEvent.change(input, { target: { files: [md] } });
    await waitFor(() => expect(onLoad).toHaveBeenCalledWith(expect.objectContaining({ fileName: "prompt.md", content: "hello" })));

    const txt = new File(["hello"], "prompt.txt", { type: "text/plain" });
    fireEvent.change(input, { target: { files: [txt] } });
    await waitFor(() => expect(onError).toHaveBeenCalledWith(expect.objectContaining({ message: "invalid" })));
  });

  it("downloads text and zip files", () => {
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});

    downloadTextFile("prompt.md", "hello");
    downloadZipFile("prompts.zip", [{ fileName: "one.md", content: "hello" }]);

    expect(click).toHaveBeenCalledTimes(2);
    click.mockRestore();
  });

  it("normalizes filenames and newlines", () => {
    expect(safeFileName("Clinical Summary V1.md")).toBe("clinical_summary_v1.md");
    expect(ensureTrailingNewline("hello")).toBe("hello\n");
  });
});
