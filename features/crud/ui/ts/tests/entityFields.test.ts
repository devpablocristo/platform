import { describe, expect, it } from "vitest";
import {
  INTERNAL_FAVORITE_FIELD_KEY,
  INTERNAL_NOTES_FIELD_KEY,
  INTERNAL_TAGS_FIELD_KEY,
  buildStandardInternalFields,
  formatTagCsv,
  parseTagCsv,
} from "../src/entityFields";

describe("buildStandardInternalFields", () => {
  it("includes favorite, tags and notes by default", () => {
    const fields = buildStandardInternalFields({ tagsPlaceholder: "demo" });
    expect(fields.map((field) => field.key)).toEqual([
      INTERNAL_FAVORITE_FIELD_KEY,
      INTERNAL_TAGS_FIELD_KEY,
      INTERNAL_NOTES_FIELD_KEY,
    ]);
    expect(fields[1]?.placeholder).toBe("demo");
  });

  it("omits favorite when includeFavorite is false", () => {
    const fields = buildStandardInternalFields({
      tagsPlaceholder: "demo",
      includeFavorite: false,
    });
    expect(fields.map((field) => field.key)).toEqual([
      INTERNAL_TAGS_FIELD_KEY,
      INTERNAL_NOTES_FIELD_KEY,
    ]);
  });

  it("omits notes when includeNotes is false", () => {
    const fields = buildStandardInternalFields({
      tagsPlaceholder: "demo",
      includeNotes: false,
    });
    expect(fields.map((field) => field.key)).toEqual([
      INTERNAL_FAVORITE_FIELD_KEY,
      INTERNAL_TAGS_FIELD_KEY,
    ]);
  });
});

describe("parseTagCsv / formatTagCsv", () => {
  it("splits csv and trims blanks", () => {
    expect(parseTagCsv(" foo,  bar ,baz,  ")).toEqual(["foo", "bar", "baz"]);
    expect(parseTagCsv("")).toEqual([]);
    expect(parseTagCsv(undefined)).toEqual([]);
    expect(parseTagCsv(false)).toEqual([]);
  });

  it("joins array with ', '", () => {
    expect(formatTagCsv(["foo", "bar"])).toBe("foo, bar");
    expect(formatTagCsv()).toBe("");
    expect(formatTagCsv([])).toBe("");
  });
});
