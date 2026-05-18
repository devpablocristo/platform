import { createClerkTokenProvider, resolveClerkBrowserConfig } from "../src/providers/clerk";

describe("clerk helpers", () => {
  it("resolves enabled config when key exists", () => {
    expect(resolveClerkBrowserConfig({ VITE_CLERK_PUBLISHABLE_KEY: "pk_test" })).toEqual({
      publishableKey: "pk_test",
      enabled: true,
    });
  });

  it("wraps clerk getToken as nullable provider", async () => {
    const provider = createClerkTokenProvider(async () => "abc");
    await expect(provider()).resolves.toBe("abc");
  });
});
