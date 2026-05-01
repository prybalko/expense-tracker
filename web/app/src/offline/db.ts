import { openDB, type IDBPDatabase } from "idb";
import type { Expense } from "../types";

export const DB_NAME = "expense-tracker";
export const DB_VERSION = 1;
export const QUEUED_WRITES = "queued_writes";
export const CACHED_EXPENSES = "cached_expenses";

export type WriteOp = "create" | "update" | "delete";

export type CreatePayload = {
  amount: number;
  description: string;
  category: string;
  date?: string;
  tempId: number;
};

export type UpdatePayload = {
  id: number;
  patch: {
    amount?: number;
    description?: string;
    category?: string;
    date?: string;
  };
};

export type DeletePayload = {
  id: number;
};

export type QueuedWrite =
  | { id?: number; op: "create"; payload: CreatePayload; createdAt: number }
  | { id?: number; op: "update"; payload: UpdatePayload; createdAt: number }
  | { id?: number; op: "delete"; payload: DeletePayload; createdAt: number };

let _dbPromise: Promise<IDBPDatabase> | null = null;

export function getDB(): Promise<IDBPDatabase> {
  if (typeof indexedDB === "undefined") {
    return Promise.reject(new Error("IndexedDB not available"));
  }
  if (!_dbPromise) {
    _dbPromise = openDB(DB_NAME, DB_VERSION, {
      upgrade(db) {
        if (!db.objectStoreNames.contains(QUEUED_WRITES)) {
          db.createObjectStore(QUEUED_WRITES, {
            keyPath: "id",
            autoIncrement: true,
          });
        }
        if (!db.objectStoreNames.contains(CACHED_EXPENSES)) {
          db.createObjectStore(CACHED_EXPENSES, { keyPath: "id" });
        }
      },
    });
  }
  return _dbPromise;
}

export async function cacheExpenses(items: Expense[], cap = 200): Promise<void> {
  try {
    const db = await getDB();
    const tx = db.transaction(CACHED_EXPENSES, "readwrite");
    const limited = items.slice(0, cap);
    await tx.store.clear();
    for (const item of limited) {
      await tx.store.put(item);
    }
    await tx.done;
  } catch {
    // ignore cache failures
  }
}

export async function readCachedExpenses(): Promise<Expense[]> {
  try {
    const db = await getDB();
    const all = (await db.getAll(CACHED_EXPENSES)) as Expense[];
    return all.sort((a, b) => (a.date < b.date ? 1 : a.date > b.date ? -1 : 0));
  } catch {
    return [];
  }
}
