export type UserStats = {
  userId: string;
  followers: number;
  followees: number;
};

export type UserListResponse = {
  userIds: string[];
  nextCursor: string;
};

export type StatsResponse = {
  users: UserStats[];
  missingUserIds: string[];
};

export const MAX_SELECTED_USERS = 100;
export const USER_PAGE_SIZE = 100;
