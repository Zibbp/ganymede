import { create } from "zustand";
import { persist } from "zustand/middleware";
import { getUserInfo, User, UserRole } from "../hooks/useAuthentication";

const roleHierarchy: { [key in UserRole]: number } = {
  [UserRole.Admin]: 4,
  [UserRole.Editor]: 3,
  [UserRole.Archiver]: 2,
  [UserRole.User]: 1,
};

interface AuthState {
  user: User | null;
  isLoggedIn: boolean;
  isLoading: boolean;
  error: string | null;

  // Actions
  fetchUser: () => Promise<void>;
  setUser: (user: User | null) => void;
  logout: () => void;
  clearError: () => void;

  // Permission checking
  hasPermission: (requiredRole: UserRole) => boolean;
  isAdmin: () => boolean;
  isEditor: () => boolean;
  isArchiver: () => boolean;
}

// Create store
const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      // Initial state
      user: null,
      isLoggedIn: false,
      isLoading: true,
      error: null,

      // Fetch user data from API
      fetchUser: async () => {
        try {
          set({ isLoading: true, error: null });
          const data = await getUserInfo();
          set({
            user: data.data,
            isLoggedIn: true,
            isLoading: false,
          });
        } catch (err) {
          set({
            error: err instanceof Error ? err.message : "Failed to fetch user",
            isLoading: false,
          });
        }
      },

      // Manually set user data
      setUser: (user) => {
        set({
          user,
          isLoggedIn: !!user,
          error: null,
        });
      },

      // Clear user data on logout
      logout: () => {
        set({
          user: null,
          isLoggedIn: false,
          error: null,
        });
      },

      // Clear any error messages
      clearError: () => {
        set({ error: null });
      },

      // Check if current user has required role permissions
      hasPermission: (requiredRole: UserRole) => {
        const { user } = get();
        if (!user) return false;
        return roleHierarchy[user.role] >= roleHierarchy[requiredRole];
      },

      // Convenience methods for common role checks
      isAdmin: () => {
        const { user } = get();
        return user?.role === UserRole.Admin;
      },

      isEditor: () => {
        const { hasPermission } = get();
        return hasPermission(UserRole.Editor);
      },

      isArchiver: () => {
        const { hasPermission } = get();
        return hasPermission(UserRole.Archiver);
      },
    }),
    {
      name: "auth-storage", // localStorage key
      // Only persist certain fields
      partialize: (state) => ({
        user: state.user,
        isLoggedIn: state.isLoggedIn,
      }),
    }
  )
);

export default useAuthStore;
