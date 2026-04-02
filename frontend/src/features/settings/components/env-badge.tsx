import { Badge } from "@/components/ui/badge";

export function EnvBadge() {
  return (
    <Badge variant="secondary" className="text-xs font-normal">
      Set via environment variable
    </Badge>
  );
}
