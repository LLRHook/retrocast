import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router";
import { useAuthStore } from "@/stores/auth";

export default function ServerAddressPage() {
  const navigate = useNavigate();
  const setServerUrl = useAuthStore((s) => s.setServerUrl);
  const savedUrl = useAuthStore((s) => s.serverUrl);
  const [url, setUrl] = useState(savedUrl || "");

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = url.trim().replace(/\/+$/, "");
    if (!trimmed) return;
    setServerUrl(trimmed);
    navigate("/login");
  }

  return (
    <form onSubmit={handleSubmit}>
      <h1 className="mb-2 text-center text-2xl font-bold text-text-primary">
        Connect to Server
      </h1>
      <p className="mb-6 text-center text-sm text-text-secondary">
        Enter the URL of your Retrocast server
      </p>
      <label className="mb-2 block text-xs font-semibold uppercase text-text-secondary">
        Server URL
      </label>
      <input
        type="url"
        value={url}
        onChange={(e) => setUrl(e.target.value)}
        placeholder="https://chat.example.com"
        required
        className="mb-6 w-full rounded bg-bg-input p-2.5 text-text-primary outline-none focus:ring-2 focus:ring-accent"
      />
      <button
        type="submit"
        className="w-full rounded bg-accent py-2.5 font-medium text-white transition-colors hover:bg-accent-hover"
      >
        Continue
      </button>
    </form>
  );
}
