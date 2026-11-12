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
  updated_at: string;
};

export type ExpensePage = {
  items: Expense[];
  nextCursor: string | null;
  serverTime: string;
};

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
