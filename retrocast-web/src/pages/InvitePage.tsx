import { useEffect, useState } from "react";
import { Link, Navigate, useNavigate, useParams } from "react-router";
import { useAuthStore } from "@/stores/auth";
import { acceptInvite } from "@/lib/invite";
import { ApiClientError } from "@/lib/api";

export default function InvitePage() {
  const { code = "" } = useParams();
  const serverUrl = useAuthStore(s => s.serverUrl);
  const authenticated = useAuthStore(s => s.isAuthenticated);
  const navigate = useNavigate();
  const [error, setError] = useState("");
  useEffect(() => {
    if (!serverUrl || !authenticated) return;
    let cancelled = false;
    acceptInvite(code).then(guild => { if (!cancelled) navigate("/app", {replace: true, state: {inviteGuildId: guild.id}}); })
      .catch(err => { if (!cancelled) setError(err instanceof ApiClientError ? err.message : "Could not connect to the server. Open the invite again to retry."); });
    return () => { cancelled = true; };
  }, [serverUrl, authenticated, code, navigate]);
  const query = `?invite_code=${encodeURIComponent(code)}`;
  if (!serverUrl) return <Navigate to={"/server" + query} replace />;
  if (!authenticated) return <Navigate to={"/register" + query} replace />;
  return <main className="p-8 text-text-primary"><h1>Join a server</h1>
    {error ? <><p role="alert">{error}</p><Link to="/app">Continue to app</Link></> : <p role="status">Joining server...</p>}
  </main>;
}
