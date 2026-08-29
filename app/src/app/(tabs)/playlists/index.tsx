import { router, useFocusEffect } from 'expo-router';
import { useCallback, useState } from 'react';
import { Alert, Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { createPlaylist, getPlaylists, type Playlist } from '@/api/client';
import { Header } from '@/components/Header';
import { PlaylistNameModal } from '@/components/PlaylistNameModal';
import { SummaryList } from '@/components/SummaryList';
import { theme } from '@/theme/theme';

export default function PlaylistsScreen() {
  const [playlists, setPlaylists] = useState<Playlist[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);

  const load = useCallback(() => {
    let ignore = false;
    getPlaylists()
      .then((data) => {
        if (!ignore) {
          setPlaylists(data);
          setError(null);
        }
      })
      .catch((err) => {
        if (!ignore) setError(err instanceof Error ? err.message : String(err));
      });
    return () => {
      ignore = true;
    };
  }, []);

  useFocusEffect(load);

  async function handleCreate(name: string) {
    setCreating(false);
    try {
      const playlist = await createPlaylist(name);
      load();
      router.push({ pathname: '/playlists/[id]', params: { id: String(playlist.id), name: playlist.name } });
    } catch (err) {
      Alert.alert('Could not create playlist', String(err));
    }
  }

  return (
    <SafeAreaView style={styles.container}>
      <Header title="Playlists" action={{ label: 'New', onPress: () => setCreating(true) }} />
      {error && (
        <View style={styles.errorBanner}>
          <Text style={styles.errorText}>{error}</Text>
        </View>
      )}
      <SummaryList
        items={playlists}
        keyExtractor={(item) => String(item.id)}
        emptyMessage="No playlists yet. Tap New to make one."
        renderRow={(item) => (
          <Pressable
            style={styles.row}
            onPress={() =>
              router.push({
                pathname: '/playlists/[id]',
                params: { id: String(item.id), name: item.name },
              })
            }
          >
            <View style={styles.rowText}>
              <Text style={styles.title} numberOfLines={1}>
                {item.name}
              </Text>
              <Text style={styles.subtitle}>
                {item.track_count} {item.track_count === 1 ? 'song' : 'songs'}
              </Text>
            </View>
          </Pressable>
        )}
      />
      <PlaylistNameModal
        visible={creating}
        title="New playlist"
        submitLabel="Create"
        onSubmit={handleCreate}
        onCancel={() => setCreating(false)}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  errorBanner: {
    padding: theme.spacing.md,
    backgroundColor: theme.colors.dangerBackground,
  },
  errorText: {
    color: theme.colors.danger,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: theme.spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.border,
  },
  rowText: {
    flex: 1,
  },
  title: {
    fontSize: theme.fontSize.lg,
  },
  subtitle: {
    fontSize: theme.fontSize.sm,
    color: theme.colors.textMuted,
    marginTop: 2,
  },
});
