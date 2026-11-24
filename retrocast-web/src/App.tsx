import { useEffect } from "react";
import { useAuthStore } from "@/stores/auth";
import { BrowserRouter, Routes, Route, Navigate } from "react-router";
import AuthLayout from "@/layouts/AuthLayout";
import AppLayout from "@/layouts/AppLayout";
import ServerAddressPage from "@/pages/ServerAddressPage";
import LoginPage from "@/pages/LoginPage";
import RegisterPage from "@/pages/RegisterPage";
import ChatArea from "@/components/ChatArea";

export default function App() {
  const restoreState = useAuthStore(s => s.restoreState);
  const restore = useAuthStore(s => s.restore);
  const authenticated = useAuthStore(s => s.isAuthenticated);
  const serverUrl = useAuthStore(s => s.serverUrl);
  useEffect(() => { void restore(); }, [restore]);
  if (restoreState !== "ready") return <main className="p-8 text-text-primary" role="status">
    {restoreState === "pending" ? "Restoring session..." : <><p>Could not connect to your server. Your session is saved.</p><button onClick={() => void restore()}>Retry</button></>}
  </main>;
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AuthLayout />}>
          <Route path="/server" element={<ServerAddressPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
        </Route>
        <Route path="/app/*" element={<AppLayout />}>
          <Route index element={<ChatArea />} />
        </Route>
        <Route path="*" element={<Navigate to={authenticated ? "/app" : serverUrl ? "/login" : "/server"} replace />} />
      </Routes>
    </BrowserRouter>
  );
}
