import type { CreateUserInput, UpdateUserInput } from "./types";

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[A-Za-z0-9-]{2,}$/u;

export function normalizeOptionalUsername(value: string): string | null {
  const username = value.trim();
  return username || null;
}

export function validateEmail(value: string): boolean {
  return EMAIL_PATTERN.test(value.trim());
}

export function buildCreateUserInput(input: {
  email: string;
  username: string;
  password: string;
}): CreateUserInput {
  const email = input.email.trim().toLocaleLowerCase();
  if (!validateEmail(email)) throw new Error("A valid email is required");
  if (input.password.length < 8) throw new Error("Password must contain at least 8 characters");
  return {
    email,
    password: input.password,
    username: normalizeOptionalUsername(input.username),
  };
}

export function buildUsernameUpdate(
  initialUsername: string | null,
  value: string
): UpdateUserInput {
  const username = normalizeOptionalUsername(value);
  if (username === initialUsername) return {};
  return { username };
}
