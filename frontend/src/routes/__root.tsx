import {
  createRootRouteWithContext,
  Outlet,
  useLocation,
} from "@tanstack/react-router";
import { lazy, Suspense } from "react";
import { NuqsAdapter } from "nuqs/adapters/tanstack-router";

import { Footer } from "@/components/footer";
import { Header } from "@/components/header";
import { Toaster } from "@/components/ui/sonner";
import { AuthProvider } from "@/contexts/auth-context";
import { ThemeProvider } from "@/contexts/theme-context";

import type { QueryClient } from "@tanstack/react-query";

const DevTools = lazy(() =>
  import("@/components/dev-tools").then((module) => ({
    default: module.DevTools,
  }))
);

interface MyRouterContext {
  queryClient: QueryClient;
}

function RootLayout() {
  const location = useLocation();
  const isLoginPage = location.pathname === "/login";

  return (
    <ThemeProvider>
      <AuthProvider>
        <NuqsAdapter>
          <div className="flex min-h-screen flex-col">
            {!isLoginPage && <Header />}
            <div className="flex-1">
              <Outlet />
            </div>
            {!isLoginPage && <Footer />}
          </div>
        </NuqsAdapter>
        <Toaster />
        {import.meta.env.DEV && (
          <Suspense fallback={null}>
            <DevTools />
          </Suspense>
        )}
      </AuthProvider>
    </ThemeProvider>
  );
}

export const Route = createRootRouteWithContext<MyRouterContext>()({
  component: RootLayout,
});
