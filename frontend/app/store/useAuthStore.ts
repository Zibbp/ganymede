import { AxiosError } from "axios";
import { create } from "zustand";
import { persist } from "zustand/middleware";
import { getUserInfo, User, UserRole } from "../hooks/useAuthentication";

// Literal keys: reading UserRole here would run during module initialization, and this module
// is in an import cycle with useAuthentication and useAxios, so the enum can still be undefined
// at that point. The mapped type keeps this exhaustive.
const roleHierarchy: { [key in UserRole]: number } = {
  admin: 4,
  editor: 3,
  archiver: 2,
  user: 1,
};

interface AuthState {
  user: User | null;
  isLoggedIn: boolean;
  isLoading: boolean;
  // False until the persisted state has been restored from localStorage (or the first
  // fetchUser call has settled). On the server and during React hydration this is always
  // false, so components can render a neutral placeholder instead of the logged-out UI,
  // which otherwise flashes briefly for logged-in users on every page load.
  hasHydrated: boolean;
  error: string | null;

  // Actions
  fetchUser: () => Promise<void>;
  setHasHydrated: (hasHydrated: boolean) => void;
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
      hasHydrated: false,
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
            hasHydrated: true,
          });
        } catch (err) {
          // 401/403 means there is no valid session: the persisted "logged in" state is stale
          const status =
            err instanceof Error && err.cause instanceof AxiosError
              ? err.cause.response?.status
              : undefined;
          if (status === 401 || status === 403) {
            set({ user: null, isLoggedIn: false, isLoading: false, hasHydrated: true, error: null });
            return;
          }
          set({
            error: err instanceof Error ? err.message : "Failed to fetch user",
            isLoading: false,
            hasHydrated: true,
          });
        }
      },

      setHasHydrated: (hasHydrated) => set({ hasHydrated }),

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
      // Runs once the persisted state has been restored on the client
      onRehydrateStorage: () => (state, error) => {
        if (error) {
          console.error("Failed to restore auth state", error);
          // On a failed restore zustand passes no state here and, for synchronous localStorage,
          // resets the store to its initial state right after this callback. Set the flag once
          // store creation has finished so it is not lost and the UI does not wait for fetchUser.
          queueMicrotask(() => useAuthStore.setState({ hasHydrated: true }));
          return;
        }
        state?.setHasHydrated(true);
      },
    }
  )
);

export default useAuthStore;
