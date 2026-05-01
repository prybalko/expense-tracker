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
import { useCategories } from "../hooks/useCategories";
import { listExpenses } from "../api/expenses";
import type { Expense } from "../types";

function toIsoDate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function parseIsoDate(s: string): Date {
  const [y, m, d] = s.split("-").map((p) => parseInt(p, 10));
  if (y && m && d) return new Date(y, m - 1, d);
  return new Date();
}

type FormProps = {
  isEdit: boolean;
  editingId: number | null;
  initialAmt: string;
  initialCategory: string;
  initialNote: string;
  initialDate: Date;
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
    const dateStr = toIsoDate(date);
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

  const fetchedExpenseQuery = useQuery<Expense | null>({
    queryKey: ["expense", editingId],
    queryFn: async () => {
      const page = await listExpenses({ limit: 50 });
      const found = page.items.find((e) => e.id === editingId);
      return found ?? null;
    },
    enabled: isEdit && !existingFromCache,
  });

  const editing: Expense | null = isEdit
    ? (existingFromCache ?? fetchedExpenseQuery.data ?? null)
    : null;

  const { data: categories = [] } = useCategories();
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

  const initialAmt = editing ? editing.amount.toFixed(2) : "";
  const initialCategory = editing ? editing.category : defaultCategory;
  const initialNote = editing ? (editing.description ?? "") : "";
  const initialDate = editing ? parseIsoDate(editing.date) : new Date();
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
      usageCounts={usageCounts}
      onClose={onClose}
    />
  );
}
