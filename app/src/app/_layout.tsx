import { Stack } from 'expo-router';

import { PlayerProvider } from '@/hooks/usePlayer';

// PlayerDock itself is rendered inside (tabs)/_layout.tsx (above the tab
// bar) and in upload.tsx (which has no tab bar to sit above) — not here,
// so it can be positioned relative to each screen's own chrome.
export default function RootLayout() {
  return (
    <PlayerProvider>
      <Stack screenOptions={{ headerShown: false }} />
    </PlayerProvider>
  );
}
