import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import CollabLockGate from "./CollabLockGate";

const mockForeign = vi.fn(() => ({
  readOnly: false,
  holderUserId: undefined as string | undefined
}));

vi.mock("@reearth/services/collab", () => ({
  useCollabLockLease: vi.fn(),
  useForeignCollabLock: () => mockForeign()
}));

vi.mock("@reearth/services/i18n/hooks", () => ({
  useT: () => (key: string, opts?: { userId?: string }) =>
    `${key}:${opts?.userId ?? ""}`
}));

describe("CollabLockGate", () => {
  beforeEach(() => {
    mockForeign.mockReturnValue({
      readOnly: false,
      holderUserId: undefined
    });
  });

  it("shows banner when peer holds lock", () => {
    mockForeign.mockReturnValueOnce({
      readOnly: true,
      holderUserId: "user-b"
    });
    render(
      <CollabLockGate resource="layer" id="layer-1">
        <div data-testid="inner">form</div>
      </CollabLockGate>
    );
    expect(screen.getByTestId("collab-lock-banner")).toHaveTextContent(
      "user-b"
    );
    expect(screen.getByTestId("inner")).toBeInTheDocument();
  });

  it("hides banner when editable", () => {
    mockForeign.mockReturnValue({
      readOnly: false,
      holderUserId: undefined
    });
    render(
      <CollabLockGate resource="widget" id="w1">
        <span>ok</span>
      </CollabLockGate>
    );
    expect(screen.queryByTestId("collab-lock-banner")).not.toBeInTheDocument();
  });
});
