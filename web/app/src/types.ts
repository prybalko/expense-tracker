export type User = {
  id: number;
  username: string;
};

export type CategoryColor = {
  bg: string;
  ink: string;
};

export type Category = {
  slug: string;
  label: string;
  icon: string;
  color: CategoryColor;
};

export type Expense = {
  id: number;
  amount: number;
  description: string;
  category: string;
  date: string;
  user_id?: number | null;
  // RFC3339Nano server timestamp. Advances on every insert / update /
  // soft-delete; the delta-sync hook diffs against it on Feed mount so the
  // Feed picks up rows changed in another tab without a full refetch.
  updated_at: string;
};

export type ExpensePage = {
  items: Expense[];
  nextCursor: string | null;
  // Server wall-clock at the time the response was assembled. The client
  // pins this as its initial lastSyncAt; subsequent Feed diffs pass it
  // back as `?since=...`.
  serverTime: string;
};

// ExpenseChanges is the payload of GET /api/expenses/changes?since=<ts>.
// `updated` covers inserts and updates uniformly (the client upserts by id);
// `deletedIds` is the tombstone list; `serverTime` is the new lastSyncAt.
export type ExpenseChanges = {
  updated: Expense[];
  deletedIds: number[];
  serverTime: string;
};

export type CategoryBreakdown = {
  category: string;
  total: number;
  count: number;
  percentage: number;
};

export type ChartPoint = {
  label: string;
  value: number;
};

export type Insights = {
  viewMode: "month" | "year";
  year: number;
  month: number;
  monthName: string;
  total: number;
  percentageChange: number;
  isIncrease: boolean;
  hasChange: boolean;
  averageSpending: number;
  averageLabel: string;
  categories: CategoryBreakdown[];
  chart: ChartPoint[];
  maxChartValue: number;
  isCurrentPeriod: boolean;
  prevYear: number;
  prevMonth: number;
  nextYear: number;
  nextMonth: number;
};

// Year-scoped counterpart to Insights for the "Year" view.
export type YearInsights = {
  year: number;
  total: number;
  averageSpending: number; // per elapsed month
  series: number[]; // 12 monthly totals, index 0 = January
  elapsedMonths: number; // 12 for a past year, current-month count for this year
  categories: CategoryBreakdown[];
  percentageChange: number;
  isIncrease: boolean;
  hasChange: boolean;
  isCurrentYear: boolean;
  prevYear: number;
};

export type CreateExpenseInput = {
  amount: number;
  description: string;
  category: string;
  date?: string;
};

export type UpdateExpenseInput = {
  amount?: number;
  description?: string;
  category?: string;
  date?: string;
};

export type InsightsQuery = {
  view: "month" | "year";
  year?: number;
  month?: number;
};
