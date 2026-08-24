import { StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { Header } from '@/components/Header';
import { theme } from '@/theme/theme';

// Stub only — no playlist schema/endpoints exist yet (see docs/PHASE-6-playlists-search.md).
export default function PlaylistsScreen() {
  return (
    <SafeAreaView style={styles.container}>
      <Header title="Playlists" />
      <View style={styles.content}>
        <Text style={styles.text}>Coming soon</Text>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  content: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  text: {
    fontSize: theme.fontSize.lg,
    color: theme.colors.textMuted,
  },
});
