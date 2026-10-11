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
          listError={query.listError}
          listLoading={query.listLoading}
          loadUsers={query.loadUsers}
          nextCursor={query.nextCursor}
          onClearSelection={query.clearSelection}
          onQuery={() => void query.querySelectedUsers()}
          onSelectUsers={query.selectUsers}
          onToggleUser={query.toggleUser}
          querying={query.querying}
          selectedUsers={query.selectedUsers}
        />
        <UserStatistics
          hasQueried={query.hasQueried}
          missingUsers={query.missingUsers}
          queryError={query.queryError}
          querying={query.querying}
          stats={query.stats}
        />
      </section>
    </main>
  );
}
