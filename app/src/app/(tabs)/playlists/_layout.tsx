import { Stack } from 'expo-router';

// Nests a Stack inside the Playlists tab so index.tsx (the playlist list)
// and [id].tsx (the drill-down) push/pop within this tab while the bottom
// tab bar stays visible — same pattern as albums/_layout.tsx.
export default function PlaylistsLayout() {
  return <Stack screenOptions={{ headerShown: false }} />;
}
