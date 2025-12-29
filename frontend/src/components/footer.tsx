import { GithubIcon } from "lucide-react";

export function Footer() {
  const currentYear = new Date().getFullYear();

  return (
    <footer className="border-t bg-background">
      <div className="container mx-auto flex h-14 items-center justify-between px-4">
        <p className="text-sm text-muted-foreground">
          {currentYear} VPS Monitor
        </p>
        <div className="flex items-center gap-4">
          <a
            href="https://github.com/hhftechnology/vps-monitor"
            target="_blank"
            rel="noopener noreferrer"
            className="text-muted-foreground hover:text-foreground transition-colors"
          >
            <GithubIcon className="size-5" />
            <span className="sr-only">GitHub</span>
          </a>
        </div>
      </div>
    </footer>
  );
}
