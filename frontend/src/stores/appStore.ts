import { create } from "zustand";

type AppStoreState = {
	initialized: boolean;
	markInitialized: () => void;
	reset: () => void;
};

const initialState = {
	initialized: false,
};

export const useAppStore = create<AppStoreState>((set) => ({
	...initialState,
	markInitialized: () => set({ initialized: true }),
	reset: () => set(initialState),
}));
