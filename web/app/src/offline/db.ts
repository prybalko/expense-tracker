import { openDB, type IDBPDatabase } from "idb";

export const DB_NAME = "expense-tracker";
export const DB_VERSION = 1;
export const QUEUED_WRITES = "queued_writes";

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
      },
    });
  }
  return _dbPromise;
}
