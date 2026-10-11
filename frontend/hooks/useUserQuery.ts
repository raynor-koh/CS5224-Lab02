"use client";

import { useCallback, useEffect, useState } from "react";
import { fetchUserStats, fetchUsers } from "@/lib/users-api";
import { MAX_SELECTED_USERS, type UserStats } from "@/types/users";

export function useUserQuery() {
  const [availableUsers, setAvailableUsers] = useState<string[]>([]);
  const [nextCursor, setNextCursor] = useState("");
  const [selectedUsers, setSelectedUsers] = useState<string[]>([]);
  const [stats, setStats] = useState<UserStats[]>([]);
  const [missingUsers, setMissingUsers] = useState<string[]>([]);
  const [listError, setListError] = useState("");
  const [queryError, setQueryError] = useState("");
  const [listLoading, setListLoading] = useState(true);
  const [querying, setQuerying] = useState(false);
  const [hasQueried, setHasQueried] = useState(false);

  const loadUsers = useCallback(async (cursor?: string) => {
    setListLoading(true);
    setListError("");
    try {
      const page = await fetchUsers(cursor);
      setAvailableUsers((current) => {
        const known = new Set(current);
        return [...current, ...page.userIds.filter((userId) => !known.has(userId))];
      });
      setNextCursor(page.nextCursor ?? "");
    } catch (cause) {
      setListError(
        cause instanceof Error
          ? cause.message
          : "Could not connect to the API. Check that the Go backend is running.",
      );
    } finally {
      setListLoading(false);
    }
  }, []);

  useEffect(() => {
    let active = true;
    void fetchUsers()
      .then((page) => {
        if (!active) return;
        setAvailableUsers(page.userIds);
        setNextCursor(page.nextCursor ?? "");
        setListError("");
      })
      .catch((cause: unknown) => {
        if (!active) return;
        setListError(cause instanceof Error ? cause.message : "Could not connect to the API. Check that the Go backend is running.");
      })
      .finally(() => {
        if (active) setListLoading(false);
      });
    return () => { active = false; };
  }, []);

  const resetResults = useCallback(() => {
    setStats([]);
    setMissingUsers([]);
    setHasQueried(false);
    setQueryError("");
  }, []);

  const toggleUser = useCallback((userId: string) => {
    setSelectedUsers((current) => {
      if (current.includes(userId)) return current.filter((selected) => selected !== userId);
      return current.length < MAX_SELECTED_USERS ? [...current, userId] : current;
    });
    resetResults();
  }, [resetResults]);

  const selectUsers = useCallback((userIds: string[]) => {
    setSelectedUsers((current) => {
      const merged = new Set(current);
      for (const userId of userIds) {
        if (merged.size >= MAX_SELECTED_USERS) break;
        merged.add(userId);
      }
      return [...merged];
    });
    resetResults();
  }, [resetResults]);

  const clearSelection = useCallback(() => {
    setSelectedUsers([]);
    resetResults();
  }, [resetResults]);

  const querySelectedUsers = useCallback(async () => {
    if (!selectedUsers.length) return;

    setQuerying(true);
    setQueryError("");
    setStats([]);
    setMissingUsers([]);
    setHasQueried(false);

    try {
      const result = await fetchUserStats(selectedUsers);
      setStats(result.users ?? []);
      setMissingUsers(result.missingUserIds ?? []);
      setHasQueried(true);
    } catch (cause) {
      setQueryError(cause instanceof Error ? cause.message : "Could not retrieve the selected users.");
    } finally {
      setQuerying(false);
    }
  }, [selectedUsers]);

  return {
    availableUsers,
    clearSelection,
    hasQueried,
    listError,
    listLoading,
    loadUsers,
    missingUsers,
    nextCursor,
    querying,
    queryError,
    querySelectedUsers,
    selectUsers,
    selectedUsers,
    stats,
    toggleUser,
  };
}
