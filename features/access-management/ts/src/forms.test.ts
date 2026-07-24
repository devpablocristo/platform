import { describe, expect, it } from "vitest";

import { buildCreateUserInput, buildUsernameUpdate } from "./forms";

describe("access user contracts", () => {
  it("requires email and preserves the password exactly", () => {
    expect(
      buildCreateUserInput({
        email: "  Person@Example.COM ",
        username: "",
        password: "  secret pass  ",
      })
    ).toEqual({
      email: "person@example.com",
      username: null,
      password: "  secret pass  ",
    });
  });

  it("rejects passwords shorter than eight characters", () => {
    expect(() =>
      buildCreateUserInput({ email: "person@example.com", username: "", password: "1234567" })
    ).toThrow(/at least 8/i);
  });

  it("omits an unchanged username and uses null to remove it", () => {
    expect(buildUsernameUpdate("person", "person")).toEqual({});
    expect(buildUsernameUpdate("person", "")).toEqual({ username: null });
  });
});
