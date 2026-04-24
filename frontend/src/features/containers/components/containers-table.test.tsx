import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ContainersTable } from "./containers-table";

const baseContainer = {
  id: "container-1",
  names: ["/api"],
  image: "ghcr.io/example/api:latest",
  image_id: "sha256:1",
  command: "node server.js",
  created: 1_700_000_000,
  state: "running",
  status: "Up 2 hours",
  host: "local",
  labels: {
    "com.docker.compose.project": "project-alpha",
  },
};

describe("ContainersTable", () => {
  it('shows "Collecting" when historical stats are not available yet', () => {
    render(
      <ContainersTable
        error={null}
        expandedGroups={[]}
        filteredContainers={[baseContainer]}
        groupBy="none"
        groupedItems={null}
        isError={false}
        isLoading={false}
        isReadOnly={false}
        onDelete={vi.fn()}
        onRetry={vi.fn()}
        onRestart={vi.fn()}
        onSelectAll={vi.fn()}
        onStart={vi.fn()}
        onStop={vi.fn()}
        onToggleGroup={vi.fn()}
        onToggleSelect={vi.fn()}
        onViewLogs={vi.fn()}
        onViewStats={vi.fn()}
        pageItems={[baseContainer]}
        pendingAction={null}
        selectedIds={[]}
        statsInterval="1h"
      />,
    );

    expect(screen.getAllByText("Collecting")).toHaveLength(2);
  });

  it("keeps the compose group label together", () => {
    render(
      <ContainersTable
        error={null}
        expandedGroups={["project-alpha"]}
        filteredContainers={[baseContainer, { ...baseContainer, id: "container-2", names: ["/worker"] }]}
        groupBy="compose"
        groupedItems={[
          {
            project: "project-alpha",
            items: [baseContainer, { ...baseContainer, id: "container-2", names: ["/worker"] }],
          },
        ]}
        isError={false}
        isLoading={false}
        isReadOnly={false}
        onDelete={vi.fn()}
        onRetry={vi.fn()}
        onRestart={vi.fn()}
        onSelectAll={vi.fn()}
        onStart={vi.fn()}
        onStop={vi.fn()}
        onToggleGroup={vi.fn()}
        onToggleSelect={vi.fn()}
        onViewLogs={vi.fn()}
        onViewStats={vi.fn()}
        pageItems={[baseContainer, { ...baseContainer, id: "container-2", names: ["/worker"] }]}
        pendingAction={null}
        selectedIds={[]}
        statsInterval="1h"
      />,
    );

    expect(screen.getByRole("button", { name: /project-alpha/i }).textContent).toContain(
      "project-alpha · 2 containers",
    );
  });

  it("renders the image copy control as an accessible button", () => {
    const writeText = vi.fn();
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });

    render(
      <ContainersTable
        error={null}
        expandedGroups={[]}
        filteredContainers={[baseContainer]}
        groupBy="none"
        groupedItems={null}
        isError={false}
        isLoading={false}
        isReadOnly={false}
        onDelete={vi.fn()}
        onRetry={vi.fn()}
        onRestart={vi.fn()}
        onSelectAll={vi.fn()}
        onStart={vi.fn()}
        onStop={vi.fn()}
        onToggleGroup={vi.fn()}
        onToggleSelect={vi.fn()}
        onViewLogs={vi.fn()}
        onViewStats={vi.fn()}
        pageItems={[baseContainer]}
        pendingAction={null}
        selectedIds={[]}
        statsInterval="1h"
      />,
    );

    fireEvent.click(
      screen.getByRole("button", { name: `Copy ${baseContainer.image}` }),
    );

    expect(writeText).toHaveBeenCalledWith(baseContainer.image);
  });
});
