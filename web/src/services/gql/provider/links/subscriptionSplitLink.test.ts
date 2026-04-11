import { describe, expect, it, vi, afterEach } from "vitest";

import { graphqlSubscriptionWsUrl } from "./subscriptionSplitLink";

describe("graphqlSubscriptionWsUrl", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("maps https GraphQL URL to wss", () => {
    expect(graphqlSubscriptionWsUrl("https://api.example.com/graphql")).toBe(
      "wss://api.example.com/graphql"
    );
  });

  it("maps http GraphQL URL to ws", () => {
    expect(graphqlSubscriptionWsUrl("http://localhost:8080/api/graphql")).toBe(
      "ws://localhost:8080/api/graphql"
    );
  });

  it("uses window host for relative path starting with /", () => {
    vi.stubGlobal("location", {
      protocol: "https:",
      host: "app.example.org"
    } as Location);
    expect(graphqlSubscriptionWsUrl("/api/graphql")).toBe(
      "wss://app.example.org/api/graphql"
    );
  });

  it("prefixes relative path without leading slash", () => {
    vi.stubGlobal("location", {
      protocol: "http:",
      host: "localhost:3000"
    } as Location);
    expect(graphqlSubscriptionWsUrl("api/graphql")).toBe(
      "ws://localhost:3000/api/graphql"
    );
  });
});
