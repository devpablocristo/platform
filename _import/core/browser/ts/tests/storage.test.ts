import { createBrowserStorageNamespace } from "../src/storage";

describe("createBrowserStorageNamespace", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("stores host-aware values", () => {
    const storage = createBrowserStorageNamespace({ namespace: "ponti" });
    storage.setString("lang", "es");

    expect(window.localStorage.getItem(`ponti:${window.location.host}:lang`)).toBe("es");
    expect(storage.getString("lang")).toBe("es");
  });

  it("stores and reads JSON values", () => {
    const storage = createBrowserStorageNamespace({ namespace: "pymes", hostAware: false });
    storage.setJSON("profile", { vertical: "workshops" });

    expect(storage.getJSON<{ vertical: string }>("profile")).toEqual({ vertical: "workshops" });
  });

  it("clears namespaced and legacy keys", () => {
    const storage = createBrowserStorageNamespace({
      namespace: "ponti",
      legacyKeys: ["access_token"],
    });
    window.localStorage.setItem("access_token", "legacy");
    window.localStorage.setItem(`ponti:${window.location.host}:lang`, "es");

    storage.clear();

    expect(window.localStorage.length).toBe(0);
  });
});
