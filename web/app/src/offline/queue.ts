import { getDB, QUEUED_WRITES, type QueuedWrite } from "./db";

export async function enqueueWrite(
  entry: Omit<QueuedWrite, "id" | "createdAt">,
): Promise<number> {
  const db = await getDB();
  const tx = db.transaction(QUEUED_WRITES, "readwrite");
  const value = { ...entry, createdAt: Date.now() } as QueuedWrite;
  const id = (await tx.store.add(value)) as IDBValidKey;
  await tx.done;
  return id as number;
}

export async function listQueued(): Promise<QueuedWrite[]> {
  const db = await getDB();
  const all = (await db.getAll(QUEUED_WRITES)) as QueuedWrite[];
  return all.sort((a, b) => a.createdAt - b.createdAt);
}

export async function removeQueued(id: number): Promise<void> {
  const db = await getDB();
  await db.delete(QUEUED_WRITES, id);
}

export async function clearQueue(): Promise<void> {
  const db = await getDB();
  await db.clear(QUEUED_WRITES);
}

export async function queueSize(): Promise<number> {
  const db = await getDB();
  return await db.count(QUEUED_WRITES);
}

export type DrainOutcome = "ok" | "drop" | "retry";
export type DrainResult = { processed: number; dropped: number; failed: number };

export async function drainQueue(
  processOne: (entry: QueuedWrite) => Promise<DrainOutcome>,
): Promise<DrainResult> {
  const items = await listQueued();
  let processed = 0;
  let dropped = 0;
  let failed = 0;
  for (const entry of items) {
    let outcome: DrainOutcome;
    try {
      outcome = await processOne(entry);
    } catch {
      // Unexpected throw from the callback — treat as transient.
      outcome = "retry";
    }
    if (outcome === "retry") {
      // Network or transient failure — stop here so the next online tick
      // picks up where we left off, preserving order.
      failed++;
      break;
    }
    if (entry.id !== undefined) {
      await removeQueued(entry.id);
    }
    if (outcome === "drop") {
      dropped++;
    } else {
      processed++;
    }
  }
  return { processed, dropped, failed };
}
