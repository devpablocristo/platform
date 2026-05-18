function hasLettersOrDigits(token: string): boolean {
  return /[\p{L}\p{N}]/u.test(token);
}

function isUppercaseAcronym(token: string): boolean {
  const alphanumeric = token.replace(/[^\p{L}\p{N}]/gu, '');
  return alphanumeric.length >= 2 && /^[A-Z0-9]+$/u.test(alphanumeric);
}

function capitalizeFirstLetter(token: string): string {
  return token.replace(/\p{L}/u, (char) => char.toLocaleUpperCase());
}

/**
 * Convierte texto a sentence case, preservando acrónimos (API, CRUD, etc.).
 */
export function toSentenceCase(text: string): string {
  let seenFirstWord = false;
  return text
    .split(/(\s+)/)
    .map((token) => {
      if (/^\s+$/u.test(token) || !hasLettersOrDigits(token)) return token;
      if (isUppercaseAcronym(token)) {
        seenFirstWord = true;
        return token;
      }
      const normalized = token.toLocaleLowerCase();
      if (!seenFirstWord) {
        seenFirstWord = true;
        return capitalizeFirstLetter(normalized);
      }
      return normalized;
    })
    .join('');
}
