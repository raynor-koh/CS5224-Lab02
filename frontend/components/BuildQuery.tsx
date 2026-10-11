"use client";

import { useMemo, useState } from "react";
import { MAX_SELECTED_USERS } from "@/types/users";

type BuildQueryProps = {
  availableUsers: string[];
  listError: string;
  listLoading: boolean;
  loadUsers: (cursor?: string) => Promise<void>;
  nextCursor: string;
  onClearSelection: () => void;
  onQuery: () => void;
  onSelectUsers: (userIds: string[]) => void;
  onToggleUser: (userId: string) => void;
  querying: boolean;
  selectedUsers: string[];
};

export function BuildQuery({
  availableUsers,
  listError,
  listLoading,
  loadUsers,
  nextCursor,
  onClearSelection,
  onQuery,
  onSelectUsers,
  onToggleUser,
  querying,
  selectedUsers,
}: BuildQueryProps) {
  const [search, setSearch] = useState("");
  const visibleUsers = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (!needle) return availableUsers;
    return availableUsers.filter((userId) => userId.toLowerCase().includes(needle));
  }, [availableUsers, search]);

  return (
    <section className="query-panel" aria-labelledby="query-heading">
      <div className="query-panel-heading">
        <div>
          <p className="eyebrow">USER SELECTION</p>
          <h2 id="query-heading">Build your query</h2>
        </div>
        <span className="selection-count">{selectedUsers.length} <span>/ {MAX_SELECTED_USERS}</span></span>
      </div>

      {listError ? (
        <div className="inline-error" role="alert">
          <span>{listError}</span>
          <button type="button" className="text-button" onClick={() => void loadUsers()}>Retry</button>
        </div>
      ) : (
        <details className="user-picker">
          <summary>
            <span>{selectedUsers.length ? `${selectedUsers.length} user${selectedUsers.length === 1 ? "" : "s"} selected` : "Select users to query"}</span>
            <span className="summary-chevron" aria-hidden="true">⌄</span>
          </summary>
          <div className="picker-content">
            <div className="picker-tools">
              <label className="search-box">
                <span aria-hidden="true">⌕</span>
                <input
                  type="search"
                  placeholder="Filter loaded user IDs"
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                />
              </label>
              <div className="picker-actions">
                <button
                  type="button"
                  className="text-button"
                  onClick={() => onSelectUsers(visibleUsers)}
                  disabled={!visibleUsers.length || selectedUsers.length >= MAX_SELECTED_USERS}
                >
                  Select visible
                </button>
                <button type="button" className="text-button" onClick={onClearSelection} disabled={!selectedUsers.length}>
                  Clear
                </button>
              </div>
            </div>

            <div className="user-options" role="group" aria-label="Available users">
              {visibleUsers.map((userId) => (
                <label className="user-option" key={userId}>
                  <input
                    type="checkbox"
                    checked={selectedUsers.includes(userId)}
                    disabled={!selectedUsers.includes(userId) && selectedUsers.length >= MAX_SELECTED_USERS}
                    onChange={() => onToggleUser(userId)}
                  />
                  <span>{userId}</span>
                </label>
              ))}
              {!visibleUsers.length && !listLoading && <p className="empty-options">No matching users in the loaded pages.</p>}
              {listLoading && availableUsers.length === 0 && <p className="empty-options">Loading available users…</p>}
            </div>

            <div className="picker-footer">
              <span>{availableUsers.length.toLocaleString()} user IDs loaded</span>
              {nextCursor ? (
                <button type="button" className="text-button" onClick={() => void loadUsers(nextCursor)} disabled={listLoading}>
                  {listLoading ? "Loading…" : "Load more users →"}
                </button>
              ) : (
                <span className="end-of-list">All user IDs loaded</span>
              )}
            </div>
          </div>
        </details>
      )}

      <div className="query-footer">
        <p>Choose up to {MAX_SELECTED_USERS} users per request.</p>
        <button className="query-button" type="button" onClick={onQuery} disabled={!selectedUsers.length || querying}>
          {querying ? "Querying…" : "Query selected users"}
          {!querying && <span aria-hidden="true">↗</span>}
        </button>
      </div>
    </section>
  );
}
