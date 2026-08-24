import { BottomTabBar, type BottomTabBarProps } from '@react-navigation/bottom-tabs';
import { Tabs } from 'expo-router';
import { Text } from 'react-native';

import { PlayerDock } from '@/components/PlayerDock';

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
  return (
    <Tabs
      screenOptions={{ headerShown: false }}
      tabBar={(props) => <TabBarWithPlayerDock {...props} />}
    >
      <Tabs.Screen
        name="index"
        options={{ title: 'Songs', tabBarIcon: () => <TabIcon glyph="🎵" /> }}
      />
      <Tabs.Screen
        name="artists"
        options={{ title: 'Artists', tabBarIcon: () => <TabIcon glyph="🎤" /> }}
      />
      <Tabs.Screen
        name="albums"
        options={{ title: 'Albums', tabBarIcon: () => <TabIcon glyph="💿" /> }}
      />
      <Tabs.Screen
        name="playlists"
        options={{ title: 'Playlists', tabBarIcon: () => <TabIcon glyph="📃" /> }}
      />
    </Tabs>
  );
}
