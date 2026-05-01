import { useEffect } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import "@fontsource/dm-sans/400.css";
import "@fontsource/dm-sans/500.css";
import "@fontsource/dm-sans/600.css";
import { Login } from "./screens/Login";
import { Feed } from "./screens/Feed";
import { Insights } from "./screens/Insights";
import { EntryForm } from "./screens/EntryForm";
import { setupOnlineSync } from "./offline/sync";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

function App() {
  useEffect(() => {
    const detach = setupOnlineSync(queryClient);
    return () => {
      detach();
    };
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/" element={<Feed />} />
          <Route path="/insights" element={<Insights />} />
          <Route path="/add" element={<EntryForm />} />
          <Route path="/edit/:id" element={<EntryForm />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default App;
