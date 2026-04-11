import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { CollabProvider } from "./CollabProvider";

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
