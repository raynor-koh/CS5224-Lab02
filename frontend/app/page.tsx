"use client";

import { BuildQuery } from "@/components/BuildQuery";
import { UserStatistics } from "@/components/UserStatistics";
import { useUserQuery } from "@/hooks/useUserQuery";

export default function Home() {
  const query = useUserQuery();

  return (
    <main className="app-shell">
      <section className="workspace" aria-label="Build your query and view user statistics">
        <BuildQuery
          availableUsers={query.availableUsers}
          hasNextPage={query.hasNextPage}
          listError={query.listError}
          listLoading={query.listLoading}
          onClearSelection={query.clearSelection}
          onNextPage={() => void query.nextPage()}
          onPreviousPage={query.previousPage}
          onQuery={() => void query.querySelectedUsers()}
          onRetryUsers={() => void query.retryUsers()}
          onSelectUsers={query.selectUsers}
          onToggleUser={query.toggleUser}
          pageNumber={query.pageNumber}
          querying={query.querying}
          selectedUsers={query.selectedUsers}
        />
        <UserStatistics
          hasQueried={query.hasQueried}
          missingUsers={query.missingUsers}
          onBrowseAll={() => void query.browseAllStats()}
          onNextPage={() => void query.nextAllStatsPage()}
          onPreviousPage={query.previousAllStatsPage}
          onRetry={() => void query.retryAllStats()}
          onShowSelected={query.showSelectedStats}
          queryError={query.queryError}
          querying={query.querying}
          showingAllStats={query.showingAllStats}
          stats={query.stats}
          statsHasNextPage={query.statsHasNextPage}
          statsPageNumber={query.statsPageNumber}
        />
      </section>
    </main>
  );
}
