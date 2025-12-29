import { redirect } from "@tanstack/react-router";

import { API_BASE_URL } from "@/types/api";

/**
 * Auth guard function that checks if authentication is enabled and redirects to login if needed.
 *
 * This function:
 * - Gets the token from localStorage
 * - If no token exists, checks if auth is enabled by calling the login endpoint
 * - Returns (allows access) when the endpoint responds with 404 (auth disabled)
 * - Redirects to /login when auth is enabled and no token is present
 * - Preserves existing catch behavior that only rethrows redirect errors
 */
export async function requireAuthIfEnabled(): Promise<void> {
  const token = localStorage.getItem("vps-monitor_auth_token");

  // If no token, check if auth is required
  if (!token) {
    try {
      const authUrl = `${API_BASE_URL}/api/v1/auth/login`.replace(
        /([^:]\/)\/+/g,
        "$1"
      );
      const response = await fetch(authUrl, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: "", password: "" }),
      });

      // If 404, auth is disabled - allow access
      if (response.status === 404) {
        return;
      }

      // Auth is enabled but no token - redirect to login
      throw redirect({ to: "/login" });
    } catch (error) {
      // If we can't reach the server, allow access (fail open for development)
      if (error instanceof Error && error.message.includes("redirect")) {
        throw error;
      }
    }
  }
}
