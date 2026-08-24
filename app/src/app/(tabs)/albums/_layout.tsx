import { Stack } from 'expo-router';

// Nests a Stack inside the Albums tab so index.tsx (the album list) and
// [name].tsx (the drill-down) push/pop within this tab while the bottom
// tab bar stays visible, per expo-router's tabs-with-nested-stack pattern.
export default function AlbumsLayout() {
  return <Stack screenOptions={{ headerShown: false }} />;
}
