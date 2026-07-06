import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { theme, FONT } from "../theme";
import { CategoryPicker } from "../components/CategoryPicker";
import { Keypad } from "../components/Keypad";
import type { KeypadKey } from "../components/Keypad";
import { DatePickerPill } from "../components/DatePickerPill";
import { useToday } from "../hooks/useToday";
import { resolveSubmitDate } from "./entryDate";
import { amountToKeypadString, applyKeypadKey } from "./entryAmount";
import {
  useAllExpenses,
  useCreateExpense,
  useUpdateExpense,
  useDeleteExpense,
} from "../hooks/useExpenses";
import { useErrorBanner } from "../hooks/useErrorBanner";
import { categories } from "../categories";
import { getExpense } from "../api/expenses";
import { messageForWriteError } from "../api/errors";
import type { Expense } from "../types";

function toIsoDateTime(d: Date): string {
  // Pin the user's calendar date and append local time-of-day with a Z
  // suffix. The server's per-user unique index on (date, amount, description)
  // would otherwise treat two legitimate same-day same-amount same-note
  // expenses (e.g. two €4.50 coffees) as a duplicate and reject the second
  // with a 409. Millisecond-precision timestamps keep them distinct. The Z
  // suffix is intentional: the system treats dates as calendar abstractions
  // (Feed slices the YYYY-MM-DD prefix), so a real timezone offset would let
  // the prefix shift across midnight UTC and break grouping.
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
  const [dateTouched, setDateTouched] = useState(false);
  const today = useToday();
  const { showError, clear: clearBannerError } = useErrorBanner();

  // The date the pill shows. For a brand-new expense the user hasn't dated
  // yet, it tracks the live current day (useToday refreshes on resume), so a
  // form left open across an iOS overnight freeze shows the real day rather
  // than the day it was opened. An explicit pick or an edit shows `date`.
  const displayedDate = resolveSubmitDate(isEdit, dateTouched, date, today);

  // Clear any lingering banner error as soon as the user starts editing
  // again — stale copy from the previous attempt should disappear rather
  // than hang around while they type. Each input handler funnels through
  // these wrappers instead of calling the raw setters directly.
  const press = (k: KeypadKey) => {
    clearBannerError();
    setAmt((a) => applyKeypadKey(a, k));
  };

  const onCatChange = (next: string) => {
    clearBannerError();
    setCat(next);
  };

  const onNoteChange = (next: string) => {
    clearBannerError();
    setNote(next);
  };

  const onDateChange = (next: Date) => {
    clearBannerError();
    setDateTouched(true);
    setDate(next);
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
    clearBannerError();
    // On edit, preserve the original full timestamp when the user hasn't
    // shifted the calendar day. Otherwise we'd silently overwrite the stored
    // time-of-day with "now" on every save (re-ordering the feed and
    // mutating the unique-index tuple even when the user only touched the
    // amount or note).
    // Last-line guarantee: a new, untouched expense persists the real current
    // day even if the resume effect hasn't flushed yet (returns `date`
    // unchanged for edits and explicit picks).
    const submitDate = resolveSubmitDate(isEdit, dateTouched, date, new Date());
    const preserveOriginal =
      isEdit && initialDateString !== null && sameCalendarDay(date, initialDate);
    const dateStr = preserveOriginal
      ? (initialDateString as string)
      : toIsoDateTime(submitDate);
    try {
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
    } catch (err) {
      showError(messageForWriteError(err));
    }
  };

  const onDelete = async () => {
    if (!isEdit || editingId === null) return;
    if (submitting) return;
    if (!window.confirm("Delete this expense?")) return;
    clearBannerError();
    try {
      await deleteMutation.mutateAsync(editingId);
      onClose();
    } catch (err) {
      showError(messageForWriteError(err));
    }
  };

  const display = amt || "0";
  const [intP, decTyped = ""] = display.split(".");
  const dotTyped = display.includes(".");
  const canSubmit = !!parseFloat(amt) && !submitting;

  return (
      <div
        data-testid="entry-form"
        style={{
          position: "relative",
          width: "100%",
          height: "100dvh",
          overflow: "hidden",
          background: t.bg,
          color: t.ink,
          fontFamily: FONT,
          display: "flex",
          flexDirection: "column",
        }}
      >
      <div
        style={{
          padding: "calc(10px + env(safe-area-inset-top)) 18px 0",
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          flexShrink: 0,
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
            color: t.ink,
            cursor: "pointer",
            display: "grid",
            placeItems: "center",
            padding: 0,
          }}
        >
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M18 6L6 18M6 6l12 12" />
          </svg>
        </button>
        <span style={{ fontSize: 16, fontWeight: 600 }}>
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
        style={{
          flex: 1,
          overflow: "hidden",
          WebkitOverflowScrolling: "touch",
          display: "flex",
          flexDirection: "column",
        }}
      >
        <div style={{ textAlign: "center", padding: "14px 0 14px", flexShrink: 0 }}>
          {/*<div*/}
          {/*  style={{*/}
          {/*    fontSize: 11,*/}
          {/*    color: t.ink2,*/}
          {/*    fontWeight: 500,*/}
          {/*    letterSpacing: "0.04em",*/}
          {/*    textTransform: "uppercase",*/}
          {/*  }}*/}
          {/*>*/}
          {/*  EUR*/}
          {/*</div>*/}
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
            {/* Cents are always shown; the untyped tail stays in the
                placeholder tone so ⌫ feedback is never hidden by padding. */}
            <span style={{ fontSize: 40 }}>
              <span style={{ color: dotTyped ? t.ink2 : t.barOther }}>.</span>
              <span style={{ color: t.ink2 }}>{decTyped.slice(0, 2)}</span>
              <span style={{ color: t.barOther }}>
                {"0".repeat(Math.max(0, 2 - decTyped.length))}
              </span>
            </span>
          </div>
        </div>

        <div style={{ flex: 1 }} aria-hidden />

        <CategoryPicker value={cat} onChange={onCatChange} usageCounts={usageCounts} />

        <div
          style={{
            display: "flex",
            gap: 8,
            padding: "6px 14px 14px",
            alignItems: "stretch",
            flexShrink: 0,
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
            <DatePickerPill
              value={displayedDate}
              onChange={onDateChange}
              today={today}
              bare
            />
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
              onChange={(e) => onNoteChange(e.target.value)}
              placeholder="e.g. Albert Heijn"
              style={{
                width: "100%",
                padding: "14px",
                borderRadius: 16,
                border: "none",
                background: t.card,
                fontSize: 16, // Font size 16px prevents iOS zooming
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
          padding: "8px 14px calc(6px + env(safe-area-inset-bottom, 16px))",
          background: t.card,
          borderTopLeftRadius: 28,
          borderTopRightRadius: 28,
          flexShrink: 0,
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

  const expensesQuery = useAllExpenses();
  const cachedExpenses = useMemo<Expense[]>(
    () => expensesQuery.data ?? [],
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

  const initialAmt = editing ? amountToKeypadString(editing.amount) : "";
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
