import { useAuth } from "../auth/AuthProvider";

export default function SaveProgressBanner() {
  const { me } = useAuth();
  if (me) return null;

  return (
    <div className="card card-warn mt-3">
      <div className="text-sm">
        <strong>Sign in</strong> to save your streak, view history, and challenge friends.
      </div>
      <div className="hstack mt-2">
        <a href="#/auth" className="btn btn-primary text-sm">Sign in</a>
        <a href="#/auth?mode=signup" className="btn btn-outline text-sm">Create account</a>
      </div>
    </div>
  );
}
