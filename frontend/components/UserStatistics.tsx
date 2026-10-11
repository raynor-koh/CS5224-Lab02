import type { UserStats } from "@/types/users";

type UserStatisticsProps = {
  hasQueried: boolean;
  missingUsers: string[];
  queryError: string;
  querying: boolean;
  stats: UserStats[];
};

export function UserStatistics({ hasQueried, missingUsers, queryError, querying, stats }: UserStatisticsProps) {
  return (
    <section className="results-section" aria-labelledby="results-heading" aria-live="polite">
      <div className="results-heading">
        <div>
          <p className="eyebrow">QUERY OUTPUT</p>
          <h2 id="results-heading">User statistics</h2>
        </div>
        {hasQueried && <span className="result-count">{stats.length} result{stats.length === 1 ? "" : "s"}</span>}
      </div>

      {queryError && <div className="result-message error" role="alert">{queryError}</div>}
      {querying && <div className="result-message">Fetching selected users…</div>}
      {!querying && !queryError && !hasQueried && (
        <div className="result-message empty-state">Select one or more users, then run your query to see their statistics.</div>
      )}
      {!querying && !queryError && hasQueried && stats.length === 0 && missingUsers.length === 0 && (
        <div className="result-message empty-state">No statistics were returned for this selection.</div>
      )}

      {hasQueried && stats.length > 0 && (
        <div className="results-list">
          {stats.map((user, index) => (
            <article className="result-user" key={user.userId}>
              <div className="result-user-heading">
                <span className="result-index">{String(index + 1).padStart(2, "0")}</span>
                <h3>{user.userId}</h3>
              </div>
              <div className="result-statements">
                <p><span className="result-id">{user.userId}</span> has <strong>{Number(user.followers).toLocaleString()}</strong> followers</p>
                <p><span className="result-id">{user.userId}</span> has <strong>{Number(user.followees).toLocaleString()}</strong> followees</p>
              </div>
            </article>
          ))}
        </div>
      )}

      {hasQueried && missingUsers.length > 0 && (
        <p className="missing-note" role="status">No stored statistics for: {missingUsers.join(", ")}</p>
      )}

      <p className="definition-note">A followee is a user followed by the selected user.</p>
    </section>
  );
}
