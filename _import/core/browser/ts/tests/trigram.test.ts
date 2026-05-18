import { describe, it, expect } from "vitest";
import { trigrams, similarity } from "../src/search/trigram";
import { search, type SearchEntry } from "../src/search/search";
import { normalize } from "../src/search/normalize";

describe("normalize", () => {
  it("lowercases and removes accents", () => {
    expect(normalize("Café")).toBe("cafe");
    expect(normalize("  HELLO  ")).toBe("hello");
    expect(normalize("Configuración")).toBe("configuracion");
  });
});

describe("trigrams", () => {
  it("generates trigrams from a word", () => {
    const result = trigrams("cat");
    expect(result.size).toBeGreaterThan(0);
    expect(result.has("cat")).toBe(true);
  });

  it("handles accents via normalization", () => {
    const a = trigrams("café");
    const b = trigrams("cafe");
    expect(a).toEqual(b);
  });

  it("returns empty-ish set for empty string", () => {
    const result = trigrams("");
    expect(result.size).toBeLessThanOrEqual(1);
  });
});

describe("similarity", () => {
  it("returns 1 for identical strings", () => {
    expect(similarity("hello", "hello")).toBe(1);
  });

  it("returns 1 for case/accent variations", () => {
    expect(similarity("Café", "cafe")).toBe(1);
  });

  it("returns high score for similar strings", () => {
    expect(similarity("producto", "productos")).toBeGreaterThan(0.7);
  });

  it("returns low score for unrelated strings", () => {
    expect(similarity("apple", "zebra")).toBeLessThan(0.3);
  });

  it("handles empty strings", () => {
    expect(similarity("", "")).toBe(1);
    expect(similarity("hello", "")).toBe(0);
    expect(similarity("", "hello")).toBe(0);
  });
});

describe("search", () => {
  type MenuItem = { label: string; to: string };

  const menu: SearchEntry<MenuItem>[] = [
    { item: { label: "Clientes", to: "/clientes" }, text: "Clientes" },
    { item: { label: "Productos", to: "/productos" }, text: "Productos" },
    { item: { label: "Ventas", to: "/ventas" }, text: "Ventas" },
    { item: { label: "Cobros", to: "/cobros" }, text: "Cobros" },
    { item: { label: "Presupuestos", to: "/presupuestos" }, text: "Presupuestos" },
    { item: { label: "Configuración", to: "/config" }, text: "Configuración" },
    { item: { label: "WhatsApp", to: "/whatsapp" }, text: "WhatsApp" },
  ];

  it("finds exact prefix match first", () => {
    const results = search("cli", menu);
    expect(results.length).toBeGreaterThan(0);
    expect(results[0].item.label).toBe("Clientes");
  });

  it("finds fuzzy match with typo", () => {
    const results = search("prodcutos", menu);
    expect(results.length).toBeGreaterThan(0);
    expect(results[0].item.label).toBe("Productos");
  });

  it("ignores accents", () => {
    const results = search("configuracion", menu);
    expect(results.length).toBeGreaterThan(0);
    expect(results[0].item.label).toBe("Configuración");
  });

  it("returns empty for unrelated query", () => {
    expect(search("zzzzzzz", menu)).toHaveLength(0);
  });

  it("returns empty for empty query", () => {
    expect(search("", menu)).toHaveLength(0);
  });

  it("handles short queries via substring boost", () => {
    const results = search("ve", menu);
    expect(results.length).toBeGreaterThan(0);
    expect(results[0].item.label).toBe("Ventas");
  });

  it("narrows results as more letters are typed", () => {
    const items: SearchEntry<{ label: string }>[] = [
      { item: { label: "Notificaciones" }, text: "Base Notificaciones" },
      { item: { label: "Notas de crédito" }, text: "Comercial Notas de crédito" },
    ];
    const r1 = search("not", items);
    expect(r1.length).toBe(2);

    const r2 = search("notif", items);
    expect(r2.length).toBe(1);
    expect(r2[0].item.label).toBe("Notificaciones");

    const r3 = search("notas", items);
    expect(r3.length).toBe(1);
    expect(r3[0].item.label).toBe("Notas de crédito");
  });

  it("finds items by word prefix when text includes section label", () => {
    const menuWithSections: SearchEntry<{ label: string }>[] = [
      { item: { label: "Notificaciones" }, text: "Operaciones Notificaciones" },
      { item: { label: "Dashboard" }, text: "General Dashboard" },
      { item: { label: "Productos" }, text: "Comercial Productos" },
    ];
    const results = search("not", menuWithSections);
    expect(results.length).toBeGreaterThan(0);
    expect(results[0].item.label).toBe("Notificaciones");
  });

  it("respects limit option", () => {
    const results = search("c", menu, { limit: 2 });
    expect(results.length).toBeLessThanOrEqual(2);
  });
});
