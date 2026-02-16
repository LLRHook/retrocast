import { BrowserRouter, Routes, Route, Navigate } from "react-router";
import AuthLayout from "@/layouts/AuthLayout";
import AppLayout from "@/layouts/AppLayout";
import ServerAddressPage from "@/pages/ServerAddressPage";
import LoginPage from "@/pages/LoginPage";
import RegisterPage from "@/pages/RegisterPage";
import ChatArea from "@/components/ChatArea";

export default function App() {
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
        <Route path="*" element={<Navigate to="/server" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
