import { useAuth } from "../auth/AuthProvider";

export default function SaveProgressBanner() {
  const { me } = useAuth();
  if (me) return null;

  return (
    <div className="rounded-lg border p-3 mt-3 bg-yellow-50">
      <div className="text-sm">
        <strong>Sign in</strong> to save your streak, view history, and challenge friends.
      </div>
      <div className="mt-2 flex gap-2">
        <a href="#/auth" className="px-3 py-1 rounded bg-black text-white text-sm">Sign in</a>
        <a href="#/auth?mode=signup" className="px-3 py-1 rounded border text-sm">Create account</a>
      </div>
    </div>
  );
}
