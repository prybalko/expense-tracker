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

export type DrainResult = { processed: number; failed: number };

export async function drainQueue(
  processOne: (entry: QueuedWrite) => Promise<void>,
): Promise<DrainResult> {
  const items = await listQueued();
  let processed = 0;
  let failed = 0;
  for (const entry of items) {
    try {
      await processOne(entry);
      if (entry.id !== undefined) {
        await removeQueued(entry.id);
      }
      processed++;
    } catch {
      failed++;
      break;
    }
  }
  return { processed, failed };
}
