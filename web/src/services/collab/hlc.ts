/** Wire shape for collab `fieldHlc` / `propertyFieldHlc` (matches server HLC JSON). */
export type CollabHlcWire = {
  wall: number;
  logical: number;
  node: string;
};

/**
 * Hybrid Logical Clock for one browser replica (LWW-register CRDT timestamps).
 * Mirrors `server/internal/collab/hlc.go` Tick/Receive semantics.
 */
export class HybridLogicalClock {
  private wall = 0;
  private logical = 0;

  constructor(readonly replicaId: string) {}

  /** Next local timestamp for an outgoing mutation. */
  tick(nowMs = Date.now()): CollabHlcWire {
    if (nowMs > this.wall) {
      this.wall = nowMs;
      this.logical = 0;
    } else {
      this.logical = (this.logical + 1) >>> 0;
    }
    return {
      wall: this.wall,
      logical: this.logical,
      node: this.replicaId
    };
  }

  /** Merge a peer/server timestamp so future local ticks are causally after `remote`. */
  receive(remote: CollabHlcWire, nowMs = Date.now()): void {
    const rp = remote.wall;
    const rl = remote.logical >>> 0;
    if (rp > this.wall) {
      this.wall = rp;
      this.logical = rl + 1;
      return;
    }
    if (rp === this.wall && rl >= this.logical) {
      this.logical = rl + 1;
      return;
    }
    if (nowMs > this.wall) {
      this.wall = nowMs;
      this.logical = 0;
      return;
    }
    if (nowMs === this.wall) {
      this.logical = (this.logical + 1) >>> 0;
      return;
    }
    this.logical = (this.logical + 1) >>> 0;
  }
}
