import { BottomTabBar, type BottomTabBarProps } from '@react-navigation/bottom-tabs';
import { Redirect, Tabs } from 'expo-router';
import { Text } from 'react-native';

import { PlayerDock } from '@/components/PlayerDock';
import { useAuth } from '@/hooks/useAuth';

function TabIcon({ glyph }: { glyph: string }) {
  return <Text style={{ fontSize: 20 }}>{glyph}</Text>;
}

// Renders the mini player directly above the native tab bar (Spotify-style
// dock) instead of below it. withSafeAreaInset={false} because BottomTabBar
// already applies the bottom safe-area inset itself.
function TabBarWithPlayerDock(props: BottomTabBarProps) {
  return (
    <>
      <PlayerDock withSafeAreaInset={false} />
      <BottomTabBar {...props} />
    </>
  );
}

export default function TabLayout() {
  const { token, isLoading } = useAuth();

  // Wait for the persisted-token check, then send logged-out users to the
  // auth stack. Hitting "/" resolves into (tabs), so this guard is what
  // makes an unauthenticated launch land on the login screen.
  if (isLoading) return null;
  if (!token) return <Redirect href="/login" />;

  return (
    <Tabs
      screenOptions={{ headerShown: false }}
      tabBar={(props) => <TabBarWithPlayerDock {...props} />}
    >
      <Tabs.Screen
        name="artists"
        options={{ title: 'Artists', tabBarIcon: () => <TabIcon glyph="🎤" /> }}
      />
      <Tabs.Screen
        name="albums"
        options={{ title: 'Albums', tabBarIcon: () => <TabIcon glyph="💿" /> }}
      />
      <Tabs.Screen
        name="index"
        options={{ title: 'Songs', tabBarIcon: () => <TabIcon glyph="🎵" /> }}
      />
      <Tabs.Screen
        name="playlists"
        options={{ title: 'Playlists', tabBarIcon: () => <TabIcon glyph="📃" /> }}
      />
    </Tabs>
  );
}
