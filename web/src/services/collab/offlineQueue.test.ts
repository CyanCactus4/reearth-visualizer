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

import localforage from "localforage";

import {
  collabOfflineDrain,
  collabOfflineFlush,
  collabOfflineNormalize,
  collabOfflinePush
} from "./offlineQueue";

const offlineStore = localforage.createInstance({
  name: "reearth-collab-offline"
});

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

describe("collabOfflineNormalize", () => {
  it("migrates legacy string[] to OfflineCollabEntry[]", async () => {
    await offlineStore.setItem("outbox:legacy", ['{"m":1}', '{"m":2}']);
    const entries = await collabOfflineNormalize("legacy");
    expect(entries).toHaveLength(2);
    expect(entries[0].raw).toBe('{"m":1}');
    expect(entries[1].raw).toBe('{"m":2}');
    expect(typeof entries[0].id).toBe("string");
    expect(typeof entries[0].ts).toBe("number");
    const stored = await offlineStore.getItem<unknown>("outbox:legacy");
    expect(Array.isArray(stored)).toBe(true);
    expect((stored as { raw: string }[])[0].raw).toBe('{"m":1}');
  });
});

describe("collabOfflineFlush", () => {
  it("removes queue when all sends succeed", async () => {
    await collabOfflinePush("flush1", "a");
    await collabOfflinePush("flush1", "b");
    await collabOfflineFlush("flush1", () => true);
    const tail = await collabOfflineNormalize("flush1");
    expect(tail).toEqual([]);
  });

  it("keeps unsent tail when trySend fails mid-queue", async () => {
    await collabOfflinePush("flush2", "x");
    await collabOfflinePush("flush2", "y");
    await collabOfflinePush("flush2", "z");
    let n = 0;
    await collabOfflineFlush("flush2", () => {
      n += 1;
      return n < 2;
    });
    const tail = await collabOfflineNormalize("flush2");
    expect(tail.map((e) => e.raw)).toEqual(["y", "z"]);
  });
});
