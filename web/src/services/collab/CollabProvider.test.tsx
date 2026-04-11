import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { CollabProvider } from "./CollabProvider";

vi.mock("@reearth/services/state", () => ({
  useNotification: () => [null, vi.fn()]
}));

vi.mock("@reearth/services/i18n/hooks", () => ({
  useT: () => (key: string) => key
}));

vi.mock("@reearth/app/lib/reearth-ui", () => ({
  Modal: ({ children }: { children?: ReactNode }) => (
    <div data-testid="mock-modal">{children}</div>
  ),
  ModalPanel: ({
    children,
    actions
  }: {
    children?: ReactNode;
    actions?: ReactNode;
  }) => (
    <div>
      {children}
      {actions}
    </div>
  ),
  Button: (props: { title?: string; onClick?: () => void }) => (
    <button type="button" title={props.title} onClick={props.onClick} />
  )
}));

vi.mock("@reearth/services/auth/useAuth", () => ({
  useAuth: () => ({
    getAccessToken: async () => "test-token",
    isAuthenticated: true,
    isLoading: false,
    error: null,
    login: vi.fn(),
    logout: vi.fn()
  })
}));

describe("CollabProvider", () => {
  it("renders children when projectId is undefined", () => {
    render(
      <CollabProvider>
        <div data-testid="child" />
      </CollabProvider>
    );
    expect(screen.getByTestId("child")).toBeInTheDocument();
  });
});
