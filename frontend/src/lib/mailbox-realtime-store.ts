import { create } from 'zustand';
import type { SSEClientState, SyncEventData } from '@/types/sse';

interface MailboxRealtimeState {
  connectionState: SSEClientState;
  isConnected: boolean;
  newEmailCount: number;
  syncStatus: Record<number, SyncEventData>;
  mailboxRefreshToken: number;
  setConnectionState: (state: SSEClientState) => void;
  incrementNewEmailCount: () => void;
  clearNewEmailCount: () => void;
  upsertSyncStatus: (data: SyncEventData) => void;
  bumpMailboxRefreshToken: () => void;
  reset: () => void;
}

export const useMailboxRealtimeStore = create<MailboxRealtimeState>((set) => ({
  connectionState: 'disconnected',
  isConnected: false,
  newEmailCount: 0,
  syncStatus: {},
  mailboxRefreshToken: 0,
  setConnectionState: (state) =>
    set({
      connectionState: state,
      isConnected: state === 'connected',
    }),
  incrementNewEmailCount: () =>
    set((state) => ({
      newEmailCount: state.newEmailCount + 1,
    })),
  clearNewEmailCount: () =>
    set({
      newEmailCount: 0,
    }),
  upsertSyncStatus: (data) =>
    set((state) => ({
      syncStatus: {
        ...state.syncStatus,
        [data.account_id]: data,
      },
    })),
  bumpMailboxRefreshToken: () =>
    set((state) => ({
      mailboxRefreshToken: state.mailboxRefreshToken + 1,
    })),
  reset: () =>
    set({
      connectionState: 'disconnected',
      isConnected: false,
      newEmailCount: 0,
      syncStatus: {},
      mailboxRefreshToken: 0,
    }),
}));
