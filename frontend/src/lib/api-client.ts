const TOKEN_KEY = "vps-monitor_auth_token";

/**
 * Authenticated fetch wrapper that automatically adds Authorization header
 * and handles 401 responses by logging out the user
 */
export async function authenticatedFetch(
  input: RequestInfo | URL,
  init?: RequestInit
): Promise<Response> {
  const token = localStorage.getItem(TOKEN_KEY);

  const headers =
    input instanceof Request ? new Headers(input.headers) : new Headers();

  if (init?.headers) {
    const initHeaders = new Headers(init.headers);
    initHeaders.forEach((value, key) => {
      headers.set(key, value);
    });
  }

  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(input, {
    ...init,
    headers,
  });

  // Handle 401 Unauthorized - token expired or invalid
  // Only redirect if auth is enabled (check by seeing if login endpoint exists)
  if (response.status === 401) {
    localStorage.removeItem(TOKEN_KEY);

    // Check if auth is enabled before redirecting
    try {
      const authCheck = await fetch(
        input.toString().replace(/\/api\/v1\/.*/, "/api/v1/auth/login"),
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ username: "", password: "" }),
        }
      );

      // Only redirect to login if auth endpoint exists (not 404)
      if (authCheck.status !== 404 && window.location.pathname !== "/login") {
        window.location.href = "/login";
      }
    } catch (error) {
      // If check fails, don't redirect
      console.error("Failed to check auth status:", error);
    }
  }

  return response;
}

/**
 * Helper to get the current auth token
 */
export function getAuthToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

/**
 * Helper to set the auth token
 */
export function setAuthToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

/**
 * Helper to remove the auth token
 */
export function removeAuthToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}
