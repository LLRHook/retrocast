import { useAuthStore } from "@/stores/auth";
import { Outlet, Navigate } from "react-router";

export default function AuthLayout() {
  const authenticated = useAuthStore(s => s.isAuthenticated);
  if (authenticated) return <Navigate to="/app" replace />;
  return (
    <div className="flex min-h-screen items-center justify-center bg-bg-tertiary">
      <div className="w-full max-w-md rounded-lg bg-bg-primary p-8 shadow-xl">
        <Outlet />
      </div>
    </div>
  );
}
