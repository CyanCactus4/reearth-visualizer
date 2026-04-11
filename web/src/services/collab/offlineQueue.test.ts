import { beforeEach, describe, expect, it, vi } from "vitest";

type Buckets = Map<string, Map<string, unknown>>;

declare global {
  // eslint-disable-next-line no-var
  var __collabOfflineTestBuckets: Buckets | undefined;
}

vi.mock("localforage", () => {
  const buckets: Buckets = new Map();
  globalThis.__collabOfflineTestBuckets = buckets;
  return {
    default: {
      createInstance: (opts: { name: string }) => {
        if (!buckets.has(opts.name)) {
          buckets.set(opts.name, new Map());
        }
        const bucket = buckets.get(opts.name)!;
        return {
          getItem: async (key: string) => bucket.get(key) ?? null,
          setItem: async (key: string, val: unknown) => {
            bucket.set(key, val);
          },
          removeItem: async (key: string) => {
            bucket.delete(key);
          }
        };
      }
    }
  };
});

import { collabOfflineDrain, collabOfflinePush } from "./offlineQueue";

beforeEach(() => {
  globalThis.__collabOfflineTestBuckets
    ?.get("reearth-collab-offline")
    ?.clear();
});

describe("collabOfflinePush / collabOfflineDrain", () => {
  it("FIFO per project and isolates projectIds", async () => {
    await collabOfflinePush("p1", '{"a":1}');
    await collabOfflinePush("p1", '{"b":2}');
    await collabOfflinePush("p2", '{"x":1}');

    const d1 = await collabOfflineDrain("p1");
    expect(d1).toEqual(['{"a":1}', '{"b":2}']);

    const d1b = await collabOfflineDrain("p1");
    expect(d1b).toEqual([]);

    const d2 = await collabOfflineDrain("p2");
    expect(d2).toEqual(['{"x":1}']);
  });
});
