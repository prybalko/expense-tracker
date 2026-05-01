import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { theme, FONT } from "../theme";
import { CategoryPicker } from "../components/CategoryPicker";
import { Keypad } from "../components/Keypad";
import type { KeypadKey } from "../components/Keypad";
import { DatePickerPill } from "../components/DatePickerPill";
import {
  useExpenses,
  useCreateExpense,
  useUpdateExpense,
  useDeleteExpense,
} from "../hooks/useExpenses";
import { categories } from "../categories";
import { getExpense } from "../api/expenses";
import type { Expense } from "../types";

function toIsoDateTime(d: Date): string {
  // Pin the user's calendar date and append local time-of-day with a Z
  // suffix. The server's per-user unique index on (date, amount, description)
  // would otherwise treat two legitimate same-day same-amount same-note
  // expenses (e.g. two €4.50 coffees) as a duplicate and the offline drain
  // would silently drop one. Millisecond-precision timestamps keep them
  // distinct; a true replay (queued payload re-posted) still carries the
  // exact stored timestamp and trips the index, so 409-driven dedupe still
  // works. The Z suffix is intentional: the system treats dates as calendar
  // abstractions (Feed slices the YYYY-MM-DD prefix), so a real timezone
  // offset would let the prefix shift across midnight UTC and break grouping.
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  const now = new Date();
  const h = String(now.getHours()).padStart(2, "0");
  const min = String(now.getMinutes()).padStart(2, "0");
  const s = String(now.getSeconds()).padStart(2, "0");
  const ms = String(now.getMilliseconds()).padStart(3, "0");
  return `${y}-${m}-${day}T${h}:${min}:${s}.${ms}Z`;
}

function parseIsoDate(s: string): Date {
  const [y, m, d] = s.split("-").map((p) => parseInt(p, 10));
  if (y && m && d) return new Date(y, m - 1, d);
  return new Date();
}

function sameCalendarDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

type FormProps = {
  isEdit: boolean;
  editingId: number | null;
  initialAmt: string;
  initialCategory: string;
  initialNote: string;
  initialDate: Date;
  // Original full timestamp string from the server, preserved across edits
  // so that re-saving an unchanged calendar date doesn't silently shift the
  // expense's stored time-of-day (and re-order it in the feed).
  initialDateString: string | null;
  usageCounts: Record<string, number>;
  onClose: () => void;
};

function FormBody({
  isEdit,
  editingId,
  initialAmt,
  initialCategory,
  initialNote,
  initialDate,
  initialDateString,
  usageCounts,
  onClose,
}: FormProps) {
  const t = theme;
  const [amt, setAmt] = useState<string>(initialAmt);
  const [cat, setCat] = useState<string>(initialCategory);
  const [note, setNote] = useState<string>(initialNote);
  const [date, setDate] = useState<Date>(initialDate);

  const press = (k: KeypadKey) => {
    setAmt((a) => {
      if (k === "del") return a.slice(0, -1);
      if (k === ".") {
        if (a.includes(".")) return a;
        if (a === "") return "0.";
        return a + ".";
      }
      if (a === "0") return k;
      const [, decPart] = a.split(".");
      if (decPart && decPart.length >= 2) return a;
      return (a + k).slice(0, 8);
    });
  };

  const createMutation = useCreateExpense();
  const updateMutation = useUpdateExpense();
  const deleteMutation = useDeleteExpense();

  const submitting =
    createMutation.isPending ||
    updateMutation.isPending ||
    deleteMutation.isPending;

  const submit = async () => {
    const v = parseFloat(amt);
    if (!v || submitting) return;
    // On edit, preserve the original full timestamp when the user hasn't
    // shifted the calendar day. Otherwise we'd silently overwrite the stored
    // time-of-day with "now" on every save (re-ordering the feed and
    // mutating the unique-index tuple even when the user only touched the
    // amount or note).
    const preserveOriginal =
      isEdit && initialDateString !== null && sameCalendarDay(date, initialDate);
    const dateStr = preserveOriginal
      ? (initialDateString as string)
      : toIsoDateTime(date);
    if (isEdit && editingId !== null) {
      await updateMutation.mutateAsync({
        id: editingId,
        patch: {
          amount: v,
          category: cat,
          description: note.trim(),
          date: dateStr,
        },
      });
    } else {
      await createMutation.mutateAsync({
        amount: v,
        category: cat,
        description: note.trim(),
        date: dateStr,
      });
    }
    onClose();
  };

  const onDelete = async () => {
    if (!isEdit || editingId === null) return;
    if (submitting) return;
    if (!window.confirm("Delete this expense?")) return;
    await deleteMutation.mutateAsync(editingId);
    onClose();
  };

  const display = amt || "0";
  const [intP, decPRaw] = display.includes(".")
    ? display.split(".")
    : [display, null];
  const decP = decPRaw === null ? null : decPRaw;
  const canSubmit = !!parseFloat(amt) && !submitting;

  return (
    <div
      data-testid="entry-form"
      style={{
        position: "relative",
        width: "100%",
        height: "100vh",
        background: t.bg,
        color: t.ink,
        fontFamily: FONT,
        display: "flex",
        flexDirection: "column",
      }}
    >
      <div
        style={{
          padding: "20px 18px 0",
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
        }}
      >
        <button
          type="button"
          onClick={onClose}
          aria-label="Close"
          style={{
            width: 36,
            height: 36,
            borderRadius: 18,
            background: t.card,
            border: "none",
            fontSize: 18,
            color: t.ink,
            cursor: "pointer",
          }}
        >
          ×
        </button>
        <span style={{ fontSize: 14, fontWeight: 500 }}>
          {isEdit ? "Edit expense" : "New expense"}
        </span>
        {isEdit ? (
          <button
            type="button"
            onClick={onDelete}
            aria-label="Delete"
            data-testid="entry-delete"
            style={{
              width: 36,
              height: 36,
              borderRadius: 18,
              background: t.card,
              border: "none",
              color: t.red,
              cursor: "pointer",
              display: "grid",
              placeItems: "center",
            }}
          >
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M6 6l1 14a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-14M10 11v6M14 11v6" />
            </svg>
          </button>
        ) : (
          <div style={{ width: 36 }} />
        )}
      </div>

      <div
        style={{ flex: 1, overflow: "auto", WebkitOverflowScrolling: "touch" }}
      >
        <div style={{ textAlign: "center", padding: "24px 0 14px" }}>
          <div
            style={{
              fontSize: 11,
              color: t.ink2,
              fontWeight: 500,
              letterSpacing: "0.04em",
              textTransform: "uppercase",
            }}
          >
            EUR
          </div>
          <div
            data-testid="entry-amount"
            data-amount={display}
            style={{
              marginTop: 6,
              fontSize: 64,
              fontWeight: 600,
              letterSpacing: "-0.03em",
              lineHeight: 1,
              fontVariantNumeric: "tabular-nums",
            }}
          >
            <span
              style={{
                fontSize: 30,
                color: amt ? t.ink2 : t.barOther,
                verticalAlign: "0.22em",
                marginRight: 2,
              }}
            >
              €
            </span>
            <span style={{ color: amt ? t.ink : t.barOther }}>{intP}</span>
            {decP !== null ? (
              <span style={{ fontSize: 40, color: t.ink2 }}>
                .{decP.padEnd(2, "0").slice(0, 2)}
              </span>
            ) : null}
          </div>
        </div>

        <CategoryPicker value={cat} onChange={setCat} usageCounts={usageCounts} />

        <div
          style={{
            display: "flex",
            gap: 8,
            padding: "6px 14px 14px",
            alignItems: "stretch",
          }}
        >
          <div style={{ flex: "0 0 46%", minWidth: 0 }}>
            <div
              style={{
                fontSize: 11,
                color: t.ink2,
                fontWeight: 500,
                letterSpacing: "0.04em",
                textTransform: "uppercase",
                padding: "0 4px 8px",
              }}
            >
              Date
            </div>
            <DatePickerPill value={date} onChange={setDate} bare />
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div
              style={{
                fontSize: 11,
                color: t.ink2,
                fontWeight: 500,
                letterSpacing: "0.04em",
                textTransform: "uppercase",
                padding: "0 4px 8px",
              }}
            >
              Note
            </div>
            <input
              data-testid="entry-note"
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="e.g. Albert Heijn"
              style={{
                width: "100%",
                padding: "14px",
                borderRadius: 16,
                border: "none",
                background: t.card,
                fontSize: 14,
                fontFamily: FONT,
                color: t.ink,
                outline: "none",
                boxSizing: "border-box",
              }}
            />
          </div>
        </div>
      </div>

      <div
        style={{
          padding: "8px 14px 28px",
          background: t.card,
          borderTopLeftRadius: 28,
          borderTopRightRadius: 28,
        }}
      >
        <Keypad onPress={press} />
        <button
          type="button"
          onClick={submit}
          disabled={!canSubmit}
          data-testid="entry-submit"
          style={{
            marginTop: 8,
            width: "100%",
            padding: "14px",
            borderRadius: 999,
            background: canSubmit ? t.accent : t.keyDisabled,
            color: t.accentText,
            border: "none",
            fontSize: 14,
            fontWeight: 600,
            cursor: canSubmit ? "pointer" : "default",
            boxShadow: canSubmit ? `0 8px 22px ${t.accent}55` : "none",
            fontFamily: FONT,
          }}
        >
          {submitting
            ? isEdit
              ? "Saving..."
              : "Adding..."
            : isEdit
              ? "Save changes"
              : "Add expense"}
        </button>
      </div>
    </div>
  );
}

export function EntryForm() {
  const navigate = useNavigate();
  const params = useParams<{ id?: string }>();
  const editingId = params.id ? parseInt(params.id, 10) : null;
  const isEdit = editingId !== null && !Number.isNaN(editingId);

  const expensesQuery = useExpenses(50);
  const cachedExpenses = useMemo<Expense[]>(
    () => expensesQuery.data?.pages.flatMap((p) => p.items) ?? [],
    [expensesQuery.data],
  );

  const existingFromCache = isEdit
    ? cachedExpenses.find((e) => e.id === editingId)
    : null;

  const fetchedExpenseQuery = useQuery<Expense>({
    queryKey: ["expense", editingId],
    queryFn: () => {
      if (editingId === null) {
        throw new Error("invalid id");
      }
      return getExpense(editingId);
    },
    enabled: isEdit && !existingFromCache,
    retry: false,
  });

  const editing: Expense | null = isEdit
    ? (existingFromCache ?? fetchedExpenseQuery.data ?? null)
    : null;

  const isFetchingFromServer =
    isEdit && !existingFromCache && fetchedExpenseQuery.isLoading;
  const fetchFailed =
    isEdit && !existingFromCache && fetchedExpenseQuery.isError;

  const defaultCategory = categories[0]?.label ?? "Other";

  const usageCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const e of cachedExpenses) {
      counts[e.category] = (counts[e.category] ?? 0) + 1;
    }
    return counts;
  }, [cachedExpenses]);

  const onClose = () => navigate(-1);

  if (isEdit && !editing) {
    if (fetchFailed) {
      return (
        <div
          style={{
            minHeight: "100vh",
            background: theme.bg,
            color: theme.ink2,
            fontFamily: FONT,
            display: "grid",
            placeItems: "center",
            padding: 24,
            textAlign: "center",
            fontSize: 13,
          }}
        >
          <div>
            <div style={{ marginBottom: 12 }}>
              Couldn't load this expense.
            </div>
            <button
              type="button"
              onClick={onClose}
              style={{
                padding: "10px 18px",
                borderRadius: 999,
                background: theme.card,
                color: theme.ink,
                border: "none",
                fontSize: 13,
                fontFamily: FONT,
                cursor: "pointer",
              }}
            >
              Go back
            </button>
          </div>
        </div>
      );
    }
    if (isFetchingFromServer) {
      return (
        <div
          style={{
            minHeight: "100vh",
            background: theme.bg,
            color: theme.ink2,
            fontFamily: FONT,
            display: "grid",
            placeItems: "center",
            fontSize: 13,
          }}
        >
          Loading...
        </div>
      );
    }
    // Edit ID resolves to no record — neither in cache nor fetched.
    return (
      <div
        style={{
          minHeight: "100vh",
          background: theme.bg,
          color: theme.ink2,
          fontFamily: FONT,
          display: "grid",
          placeItems: "center",
          padding: 24,
          textAlign: "center",
          fontSize: 13,
        }}
      >
        <div>
          <div style={{ marginBottom: 12 }}>Expense not found.</div>
          <button
            type="button"
            onClick={onClose}
            style={{
              padding: "10px 18px",
              borderRadius: 999,
              background: theme.card,
              color: theme.ink,
              border: "none",
              fontSize: 13,
              fontFamily: FONT,
              cursor: "pointer",
            }}
          >
            Go back
          </button>
        </div>
      </div>
    );
  }

  const initialAmt = editing ? editing.amount.toFixed(2) : "";
  const initialCategory = editing ? editing.category : defaultCategory;
  const initialNote = editing ? (editing.description ?? "") : "";
  const initialDate = editing ? parseIsoDate(editing.date) : new Date();
  const initialDateString = editing ? editing.date : null;
  const formKey = editing ? `edit-${editing.id}` : `new-${defaultCategory}`;

  return (
    <FormBody
      key={formKey}
      isEdit={isEdit}
      editingId={editingId}
      initialAmt={initialAmt}
      initialCategory={initialCategory}
      initialNote={initialNote}
      initialDate={initialDate}
      initialDateString={initialDateString}
      usageCounts={usageCounts}
      onClose={onClose}
    />
  );
}
