import { Stack } from 'expo-router';
import { View } from 'react-native';

import { PlayerDock } from '@/components/PlayerDock';
import { PlayerProvider } from '@/hooks/usePlayer';

export default function RootLayout() {
  return (
    <PlayerProvider>
      <View style={{ flex: 1 }}>
        <Stack screenOptions={{ headerShown: false }} />
        <PlayerDock />
      </View>
    </PlayerProvider>
  );
}
