import { Stack } from 'expo-router';

import { AuthProvider } from '@/hooks/useAuth';
import { PlayerProvider } from '@/hooks/usePlayer';

// AuthProvider wraps PlayerProvider so the token is set on the api client
// before any screen builds a stream URL from it.
//
// PlayerDock itself is rendered inside (tabs)/_layout.tsx (above the tab
// bar) and in upload.tsx (which has no tab bar to sit above) — not here,
// so it can be positioned relative to each screen's own chrome.
//
// Route protection lives in the group layouts: (auth)/_layout.tsx and
// (tabs)/_layout.tsx each <Redirect> based on useAuth(), and upload.tsx
// guards itself.
export default function RootLayout() {
  return (
    <AuthProvider>
      <PlayerProvider>
        <Stack screenOptions={{ headerShown: false }} />
      </PlayerProvider>
    </AuthProvider>
  );
}
