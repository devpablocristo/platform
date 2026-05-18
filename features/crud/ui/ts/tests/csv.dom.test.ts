// @vitest-environment jsdom

import { describe, expect, it, vi } from "vitest";
import { downloadCSVFile, pickCSVFile } from "../src/csv";

describe("csv DOM helpers", () => {
  it("downloads CSV content through a temporary anchor", () => {
    const createObjectURL = vi.fn(() => "blob:csv-url");
    const revokeObjectURL = vi.fn();
    const click = vi.fn();

    vi.stubGlobal("URL", {
      createObjectURL,
      revokeObjectURL,
    });
    HTMLAnchorElement.prototype.click = click;

    downloadCSVFile("users.csv", "name\nAda");

    expect(createObjectURL).toHaveBeenCalledTimes(1);
    expect(click).toHaveBeenCalledTimes(1);
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:csv-url");
  });

  it("picks a CSV file through a generated input", async () => {
    const file = new File(["name\nAda"], "users.csv", { type: "text/csv" });
    const realCreateElement = document.createElement.bind(document);

    vi.spyOn(document, "createElement").mockImplementation((tagName: string) => {
      const element = realCreateElement(tagName);
      if (tagName === "input") {
        const input = element as HTMLInputElement;
        Object.defineProperty(input, "files", {
          configurable: true,
          value: [file],
        });
        input.click = () => {
          input.onchange?.(new Event("change"));
        };
      }
      return element;
    });

    const picked = await pickCSVFile();
    expect(picked).toBe(file);
  });
});
