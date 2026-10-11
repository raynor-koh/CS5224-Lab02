"use client";

import { useCallback, useEffect, useState } from "react";
import { fetchAllUserStats, fetchUserStats, fetchUsers } from "@/lib/users-api";
import { MAX_SELECTED_USERS, type AllStatsPageResponse, type UserListResponse, type UserStats } from "@/types/users";

export function useUserQuery() {
  const [pages, setPages] = useState<UserListResponse[]>([]);
  const [pageIndex, setPageIndex] = useState(0);
  const [selectedUsers, setSelectedUsers] = useState<string[]>([]);
  const [stats, setStats] = useState<UserStats[]>([]);
  const [missingUsers, setMissingUsers] = useState<string[]>([]);
  const [listError, setListError] = useState("");
  const [queryError, setQueryError] = useState("");
  const [listLoading, setListLoading] = useState(true);
  const [querying, setQuerying] = useState(false);
  const [hasQueried, setHasQueried] = useState(false);
  const [statsMode, setStatsMode] = useState<"selected" | "all">("selected");
  const [allStatsPages, setAllStatsPages] = useState<AllStatsPageResponse[]>([]);
  const [allStatsPageIndex, setAllStatsPageIndex] = useState(0);
  const [allStatsLoading, setAllStatsLoading] = useState(false);
  const [allStatsError, setAllStatsError] = useState("");

  const loadFirstPage = useCallback(async () => {
    setListLoading(true);
    setListError("");
    try {
      const page = await fetchUsers();
      setPages([page]);
      setPageIndex(0);
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
        setPages([page]);
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

  const nextPage = useCallback(async () => {
    if (listLoading) return;
    if (pages[pageIndex + 1]) {
      setPageIndex(pageIndex + 1);
      setListError("");
      return;
    }

    const cursor = pages[pageIndex]?.nextCursor;
    if (!cursor) return;
    setListLoading(true);
    setListError("");
    try {
      const page = await fetchUsers(cursor);
      setPages((current) => [...current, page]);
      setPageIndex(pageIndex + 1);
    } catch (cause) {
      setListError(cause instanceof Error ? cause.message : "Could not connect to the API. Check that the Go backend is running.");
    } finally {
      setListLoading(false);
    }
  }, [listLoading, pageIndex, pages]);

  const previousPage = useCallback(() => {
    if (pageIndex === 0) return;
    setPageIndex(pageIndex - 1);
    setListError("");
  }, [pageIndex]);

  const retryUsers = pages.length === 0 ? loadFirstPage : nextPage;
  const availableUsers = pages[pageIndex]?.userIds ?? [];
  const hasNextPage = Boolean(pages[pageIndex + 1] || pages[pageIndex]?.nextCursor);
  const pageNumber = pages.length ? pageIndex + 1 : 0;

  const resetResults = useCallback(() => {
    setStatsMode("selected");
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

    setStatsMode("selected");
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

  const browseAllStats = useCallback(async () => {
    setStatsMode("all");
    if (allStatsPages.length) return;
    setAllStatsLoading(true);
    setAllStatsError("");
    try {
      const page = await fetchAllUserStats();
      setAllStatsPages([page]);
      setAllStatsPageIndex(0);
    } catch (cause) {
      setAllStatsError(cause instanceof Error ? cause.message : "Could not load user statistics.");
    } finally {
      setAllStatsLoading(false);
    }
  }, [allStatsPages.length]);

  const nextAllStatsPage = useCallback(async () => {
    if (allStatsLoading) return;
    if (allStatsPages[allStatsPageIndex + 1]) {
      setAllStatsPageIndex(allStatsPageIndex + 1);
      setAllStatsError("");
      return;
    }
    const cursor = allStatsPages[allStatsPageIndex]?.nextCursor;
    if (!cursor) return;
    setAllStatsLoading(true);
    setAllStatsError("");
    try {
      const page = await fetchAllUserStats(cursor);
      setAllStatsPages((current) => [...current, page]);
      setAllStatsPageIndex(allStatsPageIndex + 1);
    } catch (cause) {
      setAllStatsError(cause instanceof Error ? cause.message : "Could not load user statistics.");
    } finally {
      setAllStatsLoading(false);
    }
  }, [allStatsLoading, allStatsPageIndex, allStatsPages]);

  const previousAllStatsPage = useCallback(() => {
    if (allStatsPageIndex === 0) return;
    setAllStatsPageIndex(allStatsPageIndex - 1);
    setAllStatsError("");
  }, [allStatsPageIndex]);

  const showingAllStats = statsMode === "all";
  const currentAllStatsPage = allStatsPages[allStatsPageIndex];

  return {
    availableUsers,
    browseAllStats,
    clearSelection,
    hasQueried: showingAllStats ? Boolean(currentAllStatsPage) : hasQueried,
    hasNextPage,
    listError,
    listLoading,
    missingUsers: showingAllStats ? [] : missingUsers,
    nextAllStatsPage,
    nextPage,
    pageNumber,
    previousPage,
    previousAllStatsPage,
    querying: showingAllStats ? allStatsLoading : querying,
    queryError: showingAllStats ? allStatsError : queryError,
    querySelectedUsers,
    retryUsers,
    retryAllStats: allStatsPages.length ? nextAllStatsPage : browseAllStats,
    selectUsers,
    selectedUsers,
    showingAllStats,
    showSelectedStats: () => setStatsMode("selected"),
    stats: showingAllStats ? currentAllStatsPage?.users ?? [] : stats,
    statsHasNextPage: Boolean(allStatsPages[allStatsPageIndex + 1] || currentAllStatsPage?.nextCursor),
    statsPageNumber: currentAllStatsPage ? allStatsPageIndex + 1 : 0,
    toggleUser,
  };
}
