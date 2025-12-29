import { Link, useLocation } from "@tanstack/react-router";
import { BoxIcon, ImageIcon, NetworkIcon, ServerIcon } from "lucide-react";

import { ThemeToggle } from "@/components/theme-toggle";
import { Button } from "@/components/ui/button";
import { AlertBadge } from "@/features/alerts/components/alert-badge";
import { cn } from "@/lib/utils";

const navLinks = [
  { to: "/", label: "Containers", icon: BoxIcon },
  { to: "/images", label: "Images", icon: ImageIcon },
  { to: "/networks", label: "Networks", icon: NetworkIcon },
] as const;

export function Header() {
  const location = useLocation();

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto flex h-14 items-center px-4">
        <Link to="/" className="flex items-center gap-2 mr-8">
          <ServerIcon className="size-6 text-primary" />
          <span className="font-semibold text-lg">VPS Monitor</span>
        </Link>

        <nav className="flex items-center gap-1 flex-1">
          {navLinks.map((link) => {
            const isActive =
              link.to === "/"
                ? location.pathname === "/"
                : location.pathname.startsWith(link.to);
            const Icon = link.icon;

            return (
              <Button
                key={link.to}
                variant={isActive ? "secondary" : "ghost"}
                size="sm"
                asChild
                className={cn(
                  "gap-2",
                  isActive && "bg-primary/10 text-primary hover:bg-primary/20"
                )}
              >
                <Link to={link.to}>
                  <Icon className="size-4" />
                  {link.label}
                </Link>
              </Button>
            );
          })}
          <Button
            variant={location.pathname.startsWith("/alerts") ? "secondary" : "ghost"}
            size="sm"
            asChild
            className={cn(
              "gap-2",
              location.pathname.startsWith("/alerts") &&
                "bg-primary/10 text-primary hover:bg-primary/20"
            )}
          >
            <Link to="/alerts">
              <AlertBadge />
              Alerts
            </Link>
          </Button>
        </nav>

        <div className="flex items-center gap-2">
          <ThemeToggle />
        </div>
      </div>
    </header>
  );
}
