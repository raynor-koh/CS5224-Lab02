import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { UserStats } from "@/types/users";

type UserStatisticsProps = {
  hasQueried: boolean;
  missingUsers: string[];
  onBrowseAll: () => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
  onRetry: () => void;
  onShowSelected: () => void;
  queryError: string;
  querying: boolean;
  showingAllStats: boolean;
  stats: UserStats[];
  statsHasNextPage: boolean;
  statsPageNumber: number;
};

export function UserStatistics({ hasQueried, missingUsers, onBrowseAll, onNextPage, onPreviousPage,
  onRetry, onShowSelected, queryError, querying, showingAllStats, stats, statsHasNextPage,
  statsPageNumber }: UserStatisticsProps) {
  return (
    <section className="results-section" aria-labelledby="results-heading" aria-live="polite">
      <div className="results-heading">
        <div><p className="eyebrow">QUERY OUTPUT</p><h2 id="results-heading" className="section-title">User statistics</h2></div>
        <div className="results-actions">
          {hasQueried && <Badge variant="outline">{stats.length} result{stats.length === 1 ? "" : "s"}</Badge>}
          <Button type="button" variant="outline" size="sm" onClick={showingAllStats ? onShowSelected : onBrowseAll} disabled={querying}>
            {showingAllStats ? "Selected results" : "Browse all users"}
          </Button>
        </div>
      </div>
      {queryError && <Alert variant="destructive"><AlertTitle>{showingAllStats ? "Could not load statistics" : "Query failed"}</AlertTitle><AlertDescription>
        {queryError} {showingAllStats && <Button type="button" variant="outline" size="sm" onClick={onRetry}>Retry</Button>}
      </AlertDescription></Alert>}
      {querying && <Card className="results-card"><CardContent className="loading-results" aria-label={showingAllStats ? "Fetching user statistics" : "Fetching selected users"}>
        <Skeleton className="h-5 w-1/3" /><Skeleton className="h-12 w-full" /><Skeleton className="h-12 w-full" />
        <span className="sr-only">Fetching user statistics…</span>
      </CardContent></Card>}
      {!querying && !queryError && !hasQueried && <Card className="empty-card"><CardContent>{showingAllStats ? "No statistics have been loaded." : "Select one or more users, then run your query to see their statistics."}</CardContent></Card>}
      {!querying && !queryError && hasQueried && stats.length === 0 && missingUsers.length === 0 && <Card className="empty-card"><CardContent>{showingAllStats ? "No users were returned on this page." : "No statistics were returned for this selection."}</CardContent></Card>}
      {!querying && stats.length > 0 && <Card className="results-card"><CardContent className="statistics-list">
        {stats.map((user) => <article className="statistics-user" key={user.userId}>
          <p><strong>{user.userId}</strong> has <strong>{Number(user.followers).toLocaleString()}</strong> followers</p>
          <p><strong>{user.userId}</strong> has <strong>{Number(user.followees).toLocaleString()}</strong> followees</p>
        </article>)}
      </CardContent></Card>}
      {showingAllStats && statsPageNumber > 0 && <div className="stats-pagination">
        <span>Page {statsPageNumber}</span>
        <div className="picker-pagination">
          <Button type="button" variant="outline" size="sm" onClick={onPreviousPage} disabled={statsPageNumber <= 1 || querying}>Previous</Button>
          <Button type="button" variant="outline" size="sm" onClick={onNextPage} disabled={!statsHasNextPage || querying}>Next</Button>
        </div>
      </div>}
      {hasQueried && missingUsers.length > 0 && <Alert className="missing-alert" role="status"><AlertTitle>Statistics unavailable</AlertTitle><AlertDescription>No stored statistics for: {missingUsers.join(", ")}</AlertDescription></Alert>}
      <p className="definition-note">A followee is a user followed by the selected user.</p>
    </section>
  );
}
