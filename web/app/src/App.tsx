import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import "@fontsource/dm-sans/400.css";
import "@fontsource/dm-sans/500.css";
import "@fontsource/dm-sans/600.css";
import { Login } from "./screens/Login";
import { Feed } from "./screens/Feed";
import { Insights } from "./screens/Insights";
import { CategoryDetails } from "./screens/CategoryDetails";
import { EntryForm } from "./screens/EntryForm";
import { ErrorBanner } from "./components/ErrorBanner";
import { ResumeSync } from "./components/ResumeSync";
import { ErrorBannerProvider } from "./hooks/useErrorBanner";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ErrorBannerProvider>
        <ResumeSync />
        <BrowserRouter>
          <ErrorBanner />
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/" element={<Feed />} />
            <Route path="/insights" element={<Insights />} />
            <Route
              path="/insights/category/:slug"
              element={<CategoryDetails />}
            />
            <Route path="/add" element={<EntryForm />} />
            <Route path="/edit/:id" element={<EntryForm />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </ErrorBannerProvider>
    </QueryClientProvider>
  );
}

export default App;
