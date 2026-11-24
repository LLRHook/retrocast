import { useEffect, useState } from "react";
import { api } from "@/lib/api";
interface Ban { user_id: string; reason: string | null; }
export default function BansTab({guildId}: {guildId: string}) {
 const [bans,setBans] = useState<Ban[]>([]);
 const [loading,setLoading] = useState(true);
 const [error,setError] = useState("");
 const [busy,setBusy] = useState<string | null>(null);
 const [attempt,setAttempt] = useState(0);
 useEffect(() => {
  let active = true;
  api.get<Ban[]>(`/api/v1/guilds/${guildId}/bans`).then(data => {if(active) setBans(data);})
   .catch(() => {if(active) setError("Could not load bans.");})
   .finally(() => {if(active) setLoading(false);});
  return () => {active = false;};
 },[guildId,attempt]);
 async function unban(userId: string) {
  setBusy(userId); setError("");
  try {await api.delete(`/api/v1/guilds/${guildId}/bans/${userId}`);setBans(items => items.filter(b => b.user_id !== userId));}
  catch {setError("Could not unban member. Try again.");}
  finally {setBusy(null);}
 }
 return <div className="max-h-80 space-y-3 overflow-y-auto">
  {error && <div role="alert" className="text-sm text-red-400">{error}<button className="ml-2 underline" onClick={() => {setError("");setLoading(true);setAttempt(n => n + 1);}}>Retry</button></div>}
  {loading ? <p className="text-text-muted">Loading bans...</p> : !error && bans.length === 0 ? <p className="text-text-muted">No banned members.</p> : bans.map(ban => <div key={ban.user_id} className="flex items-center justify-between gap-3 rounded bg-bg-secondary p-3">
   <div className="min-w-0"><p className="break-all text-sm text-text-primary">User {ban.user_id}</p><p className="break-words text-sm text-text-muted">{ban.reason || "No reason provided"}</p></div>
   <button disabled={busy !== null} onClick={() => unban(ban.user_id)} className="rounded bg-accent px-3 py-2 text-sm text-white disabled:opacity-50">{busy === ban.user_id ? "Unbanning..." : "Unban"}</button>
  </div>)}
 </div>;
}
