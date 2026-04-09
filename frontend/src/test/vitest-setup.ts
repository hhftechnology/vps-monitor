import { expect } from "vitest";

type AsymmetricMatcher = {
  asymmetricMatch: (actual: unknown) => boolean;
};

function isAsymmetricMatcher(value: unknown): value is AsymmetricMatcher {
  return Boolean(
    value && typeof value === "object" && "asymmetricMatch" in value && typeof (value as AsymmetricMatcher).asymmetricMatch === "function"
  );
}

function matchesAttributeValue(actual: string | null, expected: unknown): boolean {
  if (expected === undefined) return actual !== null;
  if (typeof expected === "string") return actual === expected;
  if (expected instanceof RegExp) return actual !== null && expected.test(actual);
  if (isAsymmetricMatcher(expected)) return expected.asymmetricMatch(actual);
  return actual === String(expected);
}

expect.extend({
  toBeInTheDocument(received: Element | null) {
    const pass = Boolean(received && received.ownerDocument?.contains(received));
    return {
      pass,
      message: () =>
        pass
          ? "expected element not to be present in the document"
          : "expected element to be present in the document",
    };
  },
  toHaveAttribute(received: Element | null, name: string, expected?: unknown) {
    const actual = received?.getAttribute(name) ?? null;
    const pass = matchesAttributeValue(actual, expected);
    return {
      pass,
      message: () =>
        pass
          ? `expected element not to have attribute ${name}`
          : `expected element to have attribute ${name}`,
    };
  },
});

declare module "vitest" {
  interface Assertion<T = any> {
    toBeInTheDocument(): T;
    toHaveAttribute(name: string, expected?: unknown): T;
  }

  interface AsymmetricMatchersContaining {
    toBeInTheDocument(): void;
    toHaveAttribute(name: string, expected?: unknown): void;
  }
}
