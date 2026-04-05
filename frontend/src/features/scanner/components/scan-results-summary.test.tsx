import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { SeveritySummary } from "../types";
import { ScanResultsSummary } from "./scan-results-summary";

function makeSummary(overrides: Partial<SeveritySummary> = {}): SeveritySummary {
  return {
    critical: 0,
    high: 0,
    medium: 0,
    low: 0,
    negligible: 0,
    unknown: 0,
    total: 0,
    ...overrides,
  };
}

describe("ScanResultsSummary", () => {
  it("shows 'No vulnerabilities' badge when total is zero", () => {
    render(<ScanResultsSummary summary={makeSummary({ total: 0 })} />);
    expect(screen.getByText("No vulnerabilities")).toBeInTheDocument();
  });

  it("does not show severity badges when all counts are zero", () => {
    render(<ScanResultsSummary summary={makeSummary()} />);
    expect(screen.queryByText(/Critical/)).not.toBeInTheDocument();
    expect(screen.queryByText(/High/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Medium/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Low/)).not.toBeInTheDocument();
  });

  it("shows Critical badge when critical count is non-zero", () => {
    render(<ScanResultsSummary summary={makeSummary({ critical: 3, total: 3 })} />);
    expect(screen.getByText("3 Critical")).toBeInTheDocument();
  });

  it("shows High badge when high count is non-zero", () => {
    render(<ScanResultsSummary summary={makeSummary({ high: 5, total: 5 })} />);
    expect(screen.getByText("5 High")).toBeInTheDocument();
  });

  it("shows Medium badge when medium count is non-zero", () => {
    render(<ScanResultsSummary summary={makeSummary({ medium: 2, total: 2 })} />);
    expect(screen.getByText("2 Medium")).toBeInTheDocument();
  });

  it("shows Low badge when low count is non-zero", () => {
    render(<ScanResultsSummary summary={makeSummary({ low: 10, total: 10 })} />);
    expect(screen.getByText("10 Low")).toBeInTheDocument();
  });

  it("shows multiple severity badges simultaneously", () => {
    render(
      <ScanResultsSummary
        summary={makeSummary({ critical: 1, high: 2, medium: 3, low: 4, total: 10 })}
      />
    );
    expect(screen.getByText("1 Critical")).toBeInTheDocument();
    expect(screen.getByText("2 High")).toBeInTheDocument();
    expect(screen.getByText("3 Medium")).toBeInTheDocument();
    expect(screen.getByText("4 Low")).toBeInTheDocument();
  });

  it("does not show 'No vulnerabilities' when total is non-zero", () => {
    render(<ScanResultsSummary summary={makeSummary({ high: 1, total: 1 })} />);
    expect(screen.queryByText("No vulnerabilities")).not.toBeInTheDocument();
  });

  it("does not show badge for zero critical count even if others are non-zero", () => {
    render(<ScanResultsSummary summary={makeSummary({ high: 1, total: 1 })} />);
    expect(screen.queryByText(/Critical/)).not.toBeInTheDocument();
  });
});