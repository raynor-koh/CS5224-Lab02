import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
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
        <div><p className="eyebrow">QUERY OUTPUT</p><h2 id="results-heading" className="section-title">User statistics</h2></div>
        {hasQueried && <Badge variant="outline">{stats.length} result{stats.length === 1 ? "" : "s"}</Badge>}
      </div>
      {queryError && <Alert variant="destructive"><AlertTitle>Query failed</AlertTitle><AlertDescription>{queryError}</AlertDescription></Alert>}
      {querying && <Card className="results-card"><CardContent className="loading-results" aria-label="Fetching selected users">
        <Skeleton className="h-5 w-1/3" /><Skeleton className="h-12 w-full" /><Skeleton className="h-12 w-full" />
        <span className="sr-only">Fetching selected users…</span>
      </CardContent></Card>}
      {!querying && !queryError && !hasQueried && <Card className="empty-card"><CardContent>Select one or more users, then run your query to see their statistics.</CardContent></Card>}
      {!querying && !queryError && hasQueried && stats.length === 0 && missingUsers.length === 0 && <Card className="empty-card"><CardContent>No statistics were returned for this selection.</CardContent></Card>}
      {!querying && stats.length > 0 && <Card className="results-card"><CardContent className="results-table-wrap">
        <Table>
          <TableHeader><TableRow><TableHead scope="col">User ID</TableHead><TableHead scope="col" className="numeric-cell">Followers</TableHead><TableHead scope="col" className="numeric-cell">Followees</TableHead></TableRow></TableHeader>
          <TableBody>{stats.map((user) => <TableRow key={user.userId}>
            <TableCell className="user-id-cell" title={user.userId}>{user.userId}</TableCell>
            <TableCell className="numeric-cell">{Number(user.followers).toLocaleString()}</TableCell>
            <TableCell className="numeric-cell">{Number(user.followees).toLocaleString()}</TableCell>
          </TableRow>)}</TableBody>
        </Table>
      </CardContent></Card>}
      {hasQueried && missingUsers.length > 0 && <Alert className="missing-alert" role="status"><AlertTitle>Statistics unavailable</AlertTitle><AlertDescription>No stored statistics for: {missingUsers.join(", ")}</AlertDescription></Alert>}
      <p className="definition-note">A followee is a user followed by the selected user.</p>
    </section>
  );
}
