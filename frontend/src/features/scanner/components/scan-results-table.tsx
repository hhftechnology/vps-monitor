import { useMemo, useState } from "react";
import { ArrowUpDownIcon, ExternalLinkIcon, SearchIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

import type { SeverityLevel, Vulnerability } from "../types";

interface ScanResultsTableProps {
  vulnerabilities: Vulnerability[];
}

const severityOrder: Record<SeverityLevel, number> = {
  Critical: 0,
  High: 1,
  Medium: 2,
  Low: 3,
  Negligible: 4,
  Unknown: 5,
};

const severityColors: Record<SeverityLevel, string> = {
  Critical: "bg-red-600 text-white hover:bg-red-600",
  High: "bg-red-500 text-white hover:bg-red-500",
  Medium: "bg-orange-500 text-white hover:bg-orange-500",
  Low: "bg-yellow-500 text-white hover:bg-yellow-500",
  Negligible: "bg-gray-400 text-white hover:bg-gray-400",
  Unknown: "bg-gray-300 text-gray-700 hover:bg-gray-300",
};

type SortField = "severity" | "id" | "package";
type SortDir = "asc" | "desc";

export function ScanResultsTable({ vulnerabilities }: ScanResultsTableProps) {
  const [search, setSearch] = useState("");
  const [sortField, setSortField] = useState<SortField>("severity");
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDir("asc");
    }
  };

  const filtered = useMemo(() => {
    let items = vulnerabilities;
    if (search) {
      const s = search.toLowerCase();
      items = items.filter(
        (v) =>
          v.id.toLowerCase().includes(s) ||
          v.package.toLowerCase().includes(s) ||
          v.severity.toLowerCase().includes(s)
      );
    }

    items = [...items].sort((a, b) => {
      let cmp = 0;
      switch (sortField) {
        case "severity":
          cmp = severityOrder[a.severity] - severityOrder[b.severity];
          break;
        case "id":
          cmp = a.id.localeCompare(b.id);
          break;
        case "package":
          cmp = a.package.localeCompare(b.package);
          break;
      }
      return sortDir === "asc" ? cmp : -cmp;
    });

    return items;
  }, [vulnerabilities, search, sortField, sortDir]);

  return (
    <div className="space-y-3">
      <div className="relative">
        <SearchIcon className="absolute left-2 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
        <Input
          placeholder="Filter vulnerabilities..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-8"
        />
      </div>

      <div className="max-h-[400px] overflow-auto rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-auto p-0 font-medium"
                  onClick={() => toggleSort("id")}
                >
                  CVE ID <ArrowUpDownIcon className="ml-1 size-3" />
                </Button>
              </TableHead>
              <TableHead>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-auto p-0 font-medium"
                  onClick={() => toggleSort("severity")}
                >
                  Severity <ArrowUpDownIcon className="ml-1 size-3" />
                </Button>
              </TableHead>
              <TableHead>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-auto p-0 font-medium"
                  onClick={() => toggleSort("package")}
                >
                  Package <ArrowUpDownIcon className="ml-1 size-3" />
                </Button>
              </TableHead>
              <TableHead>Installed</TableHead>
              <TableHead>Fixed in</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-muted-foreground py-6">
                  {search ? "No matching vulnerabilities" : "No vulnerabilities found"}
                </TableCell>
              </TableRow>
            ) : (
              filtered.map((vuln, index) => (
                <TableRow key={`${vuln.id}-${vuln.package}-${index}`}>
                  <TableCell>
                    {vuln.id.startsWith("CVE-") ? (
                      <a
                        href={`https://nvd.nist.gov/vuln/detail/${vuln.id}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-sm hover:underline"
                      >
                        {vuln.id}
                        <ExternalLinkIcon className="size-3" />
                      </a>
                    ) : vuln.id.startsWith("GHSA-") ? (
                      <a
                        href={`https://github.com/advisories/${vuln.id}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-sm hover:underline"
                      >
                        {vuln.id}
                        <ExternalLinkIcon className="size-3" />
                      </a>
                    ) : (
                      <span className="inline-flex items-center text-sm">
                        {vuln.id}
                      </span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge className={severityColors[vuln.severity]}>
                      {vuln.severity.toLowerCase()}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono text-sm">{vuln.package}</TableCell>
                  <TableCell className="font-mono text-sm">{vuln.installed_version}</TableCell>
                  <TableCell className="font-mono text-sm">
                    {vuln.fixed_version ? (
                      <span className="text-green-600">{vuln.fixed_version}</span>
                    ) : (
                      <span className="text-muted-foreground">-</span>
                    )}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <p className="text-xs text-muted-foreground">
        Showing {filtered.length} of {vulnerabilities.length} vulnerabilities
      </p>
    </div>
  );
}
