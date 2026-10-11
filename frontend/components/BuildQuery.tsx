"use client";

import { useMemo, useState } from "react";
import { ArrowUpRight, ChevronDown, Search } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardAction, CardContent, CardDescription, CardHeader } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
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

export function BuildQuery({ availableUsers, listError, listLoading, loadUsers, nextCursor,
  onClearSelection, onQuery, onSelectUsers, onToggleUser, querying, selectedUsers }: BuildQueryProps) {
  const [search, setSearch] = useState("");
  const visibleUsers = useMemo(() => {
    const needle = search.trim().toLowerCase();
    return needle ? availableUsers.filter((id) => id.toLowerCase().includes(needle)) : availableUsers;
  }, [availableUsers, search]);
  const selectedSet = useMemo(() => new Set(selectedUsers), [selectedUsers]);

  return (
    <Card className="query-card" aria-labelledby="query-heading">
      <CardHeader className="query-header">
        <div>
          <p className="eyebrow">USER SELECTION</p>
          <h2 id="query-heading" className="section-title">Build your query</h2>
          <CardDescription className="section-description">Choose users in the social graph to compare their connections.</CardDescription>
        </div>
        <CardAction><Badge variant="secondary" className="count-badge" aria-label={`${selectedUsers.length} of ${MAX_SELECTED_USERS} users selected`}>
          {selectedUsers.length} <span>/ {MAX_SELECTED_USERS}</span>
        </Badge></CardAction>
      </CardHeader>
      <CardContent>
        {listError ? (
          <Alert variant="destructive" className="mb-5">
            <AlertTitle>Could not load users</AlertTitle>
            <AlertDescription className="flex flex-wrap items-center justify-between gap-2">
              <span>{listError}</span>
              <Button type="button" variant="outline" size="sm" onClick={() => void loadUsers()}>Retry</Button>
            </AlertDescription>
          </Alert>
        ) : (
          <Collapsible className="user-picker">
            <CollapsibleTrigger asChild>
              <Button type="button" variant="outline" className="picker-trigger">
                <span>{selectedUsers.length ? `${selectedUsers.length} user${selectedUsers.length === 1 ? "" : "s"} selected` : "Select users to query"}</span>
                <ChevronDown className="picker-chevron size-4 text-muted-foreground" aria-hidden="true" />
              </Button>
            </CollapsibleTrigger>
            <CollapsibleContent className="picker-content">
              <div className="picker-toolbar">
                <div className="search-field">
                  <Search className="size-4 text-muted-foreground" aria-hidden="true" />
                  <Input type="search" aria-label="Filter loaded user IDs" placeholder="Filter loaded user IDs"
                    value={search} onChange={(event) => setSearch(event.target.value)} />
                </div>
                <div className="picker-actions">
                  <Button type="button" size="sm" variant="ghost" onClick={() => onSelectUsers(visibleUsers)} disabled={!visibleUsers.length || selectedUsers.length >= MAX_SELECTED_USERS}>Select visible</Button>
                  <Button type="button" size="sm" variant="ghost" onClick={onClearSelection} disabled={!selectedUsers.length}>Clear</Button>
                </div>
              </div>
              <div className="user-options" role="group" aria-label="Available users">
                {visibleUsers.map((userId, index) => {
                  const checked = selectedSet.has(userId);
                  const optionId = `user-option-${index}`;
                  return <div className="user-option" key={userId}>
                    <Checkbox id={optionId} checked={checked} disabled={!checked && selectedUsers.length >= MAX_SELECTED_USERS}
                      onCheckedChange={() => onToggleUser(userId)} />
                    <label htmlFor={optionId} className="user-option-label">{userId}</label>
                  </div>;
                })}
                {!visibleUsers.length && !listLoading && <p className="option-message">No matching users in the loaded pages.</p>}
                {listLoading && availableUsers.length === 0 && <div className="loading-options"><Skeleton className="h-5 w-2/3" /><Skeleton className="h-5 w-1/2" /><Skeleton className="h-5 w-3/4" /></div>}
              </div>
              <div className="picker-footer">
                <span>{availableUsers.length.toLocaleString()} user IDs loaded</span>
                {nextCursor ? <Button type="button" variant="link" size="sm" onClick={() => void loadUsers(nextCursor)} disabled={listLoading}>
                  {listLoading ? "Loading…" : "Load more users →"}
                </Button> : <span>All user IDs loaded</span>}
              </div>
            </CollapsibleContent>
          </Collapsible>
        )}
        <div className="query-footer">
          <p>Choose up to {MAX_SELECTED_USERS} users per request.</p>
          <Button type="button" size="lg" onClick={onQuery} disabled={!selectedUsers.length || querying}>
            {querying ? "Querying…" : "Query selected users"}
            {!querying && <ArrowUpRight className="size-4" aria-hidden="true" />}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
