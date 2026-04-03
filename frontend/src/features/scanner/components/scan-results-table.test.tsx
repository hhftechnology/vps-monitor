import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { Vulnerability } from "../types";
import { ScanResultsTable } from "./scan-results-table";

function makeVuln(overrides: Partial<Vulnerability> = {}): Vulnerability {
  return {
    id: "CVE-2023-0001",
    severity: "High",
    package: "openssl",
    installed_version: "1.1.1t",
    fixed_version: "1.1.1u",
    ...overrides,
  };
}

const vulns: Vulnerability[] = [
  makeVuln({ id: "CVE-2023-0001", severity: "Critical", package: "libssl" }),
  makeVuln({ id: "CVE-2023-0002", severity: "High", package: "curl" }),
  makeVuln({ id: "CVE-2023-0003", severity: "Medium", package: "zlib" }),
  makeVuln({ id: "CVE-2023-0004", severity: "Low", package: "bash" }),
];

describe("ScanResultsTable", () => {
  it("renders all vulnerabilities", () => {
    render(<ScanResultsTable vulnerabilities={vulns} />);
    for (const v of vulns) {
      expect(screen.getByText(v.id)).toBeInTheDocument();
    }
  });

  it("shows 'No vulnerabilities found' when list is empty", () => {
    render(<ScanResultsTable vulnerabilities={[]} />);
    expect(screen.getByText("No vulnerabilities found")).toBeInTheDocument();
  });

  it("shows total and filtered counts", () => {
    render(<ScanResultsTable vulnerabilities={vulns} />);
    // Should show "Showing 4 of 4 vulnerabilities"
    expect(screen.getByText(/Showing 4 of 4/)).toBeInTheDocument();
  });

  it("filters by CVE ID when typing in search box", () => {
    render(<ScanResultsTable vulnerabilities={vulns} />);

    const searchInput = screen.getByPlaceholderText("Filter vulnerabilities...");
    fireEvent.change(searchInput, { target: { value: "CVE-2023-0001" } });

    expect(screen.getByText("CVE-2023-0001")).toBeInTheDocument();
    expect(screen.queryByText("CVE-2023-0002")).not.toBeInTheDocument();
    expect(screen.getByText(/Showing 1 of 4/)).toBeInTheDocument();
  });

  it("filters by package name when typing in search box", () => {
    render(<ScanResultsTable vulnerabilities={vulns} />);

    const searchInput = screen.getByPlaceholderText("Filter vulnerabilities...");
    fireEvent.change(searchInput, { target: { value: "curl" } });

    expect(screen.getByText("CVE-2023-0002")).toBeInTheDocument();
    expect(screen.queryByText("CVE-2023-0001")).not.toBeInTheDocument();
  });

  it("filters by severity when typing (case-insensitive)", () => {
    render(<ScanResultsTable vulnerabilities={vulns} />);

    const searchInput = screen.getByPlaceholderText("Filter vulnerabilities...");
    fireEvent.change(searchInput, { target: { value: "medium" } });

    expect(screen.getByText("CVE-2023-0003")).toBeInTheDocument();
    expect(screen.queryByText("CVE-2023-0001")).not.toBeInTheDocument();
  });

  it("shows 'No matching vulnerabilities' when search has no results", () => {
    render(<ScanResultsTable vulnerabilities={vulns} />);

    const searchInput = screen.getByPlaceholderText("Filter vulnerabilities...");
    fireEvent.change(searchInput, { target: { value: "xyznotexist" } });

    expect(screen.getByText("No matching vulnerabilities")).toBeInTheDocument();
  });

  it("renders fixed_version when available", () => {
    const vuln = makeVuln({ fixed_version: "2.0.0" });
    render(<ScanResultsTable vulnerabilities={[vuln]} />);
    expect(screen.getByText("2.0.0")).toBeInTheDocument();
  });

  it("renders dash placeholder when fixed_version is absent", () => {
    const vuln = makeVuln({ fixed_version: undefined });
    render(<ScanResultsTable vulnerabilities={[vuln]} />);
    expect(screen.getByText("-")).toBeInTheDocument();
  });

  it("renders CVE link pointing to NVD", () => {
    render(<ScanResultsTable vulnerabilities={[makeVuln({ id: "CVE-2023-9999" })]} />);
    const link = screen.getByRole("link", { name: /CVE-2023-9999/ });
    expect(link).toHaveAttribute("href", expect.stringContaining("CVE-2023-9999"));
  });

  it("renders package name in each row", () => {
    render(<ScanResultsTable vulnerabilities={vulns} />);
    expect(screen.getByText("libssl")).toBeInTheDocument();
    expect(screen.getByText("curl")).toBeInTheDocument();
  });

  it("sorts by severity ascending by default (Critical first)", () => {
    const mixed = [
      makeVuln({ id: "LOW-1", severity: "Low", package: "pkg-a" }),
      makeVuln({ id: "CRIT-1", severity: "Critical", package: "pkg-b" }),
      makeVuln({ id: "HIGH-1", severity: "High", package: "pkg-c" }),
    ];
    render(<ScanResultsTable vulnerabilities={mixed} />);

    const rows = screen.getAllByRole("row");
    // Header row + data rows; first data row should be Critical
    const firstDataRow = rows[1];
    expect(firstDataRow.textContent).toContain("CRIT-1");
  });

  it("shows installed_version in each row", () => {
    const vuln = makeVuln({ installed_version: "3.0.0-beta" });
    render(<ScanResultsTable vulnerabilities={[vuln]} />);
    expect(screen.getByText("3.0.0-beta")).toBeInTheDocument();
  });
});