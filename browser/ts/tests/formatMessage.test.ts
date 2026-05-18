// @vitest-environment node
import { describe, expect, it } from "vitest";
import { formatMessage, interpolate } from "../src/i18n/formatMessage";
import type { FlatMessages, LanguageCode } from "../src/i18n/types";

const messages: Record<LanguageCode, FlatMessages> = {
  en: { "a.b": "Hello {name}", "x": "v1" },
  es: { "a.b": "Hola {name}", "x": "v2" },
};

describe("interpolate", () => {
  it("reemplaza {var}", () => {
    expect(interpolate("Hi {x}", { x: "1" })).toBe("Hi 1");
  });

  it("normaliza {{var}} a {var}", () => {
    expect(interpolate("Hi {{x}}", { x: "1" })).toBe("Hi 1");
  });
});

describe("formatMessage", () => {
  it("usa idioma activo y cae al default", () => {
    expect(formatMessage(messages, "en", "en", "a.b", { name: "Pat" })).toBe("Hello Pat");
    expect(formatMessage(messages, "es", "en", "a.b", { name: "Pat" })).toBe("Hola Pat");
    expect(formatMessage(messages, "es", "en", "missing")).toBe("missing");
  });

  it("usa default cuando falta clave en idioma activo", () => {
    const m: Record<LanguageCode, FlatMessages> = {
      en: { only: "EN" },
      es: {},
    };
    expect(formatMessage(m, "es", "en", "only")).toBe("EN");
  });
});
