import { router, useFocusEffect } from 'expo-router';
import { useCallback, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { getTracks, type Track } from '@/api/client';
import { AddToPlaylistSheet } from '@/components/AddToPlaylistSheet';
import { Header } from '@/components/Header';
import { TrackList } from '@/components/TrackList';
import { usePlayer } from '@/hooks/usePlayer';
import { theme } from '@/theme/theme';

export default function SongsScreen() {
  const [tracks, setTracks] = useState<Track[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [addTarget, setAddTarget] = useState<Track | null>(null);
  const { playQueue, playingTrack } = usePlayer();

  // Re-fetches every time this screen gains focus (including on return
  // from the upload screen), following React's documented fetch pattern
  // (https://react.dev/learn/synchronizing-with-effects#fetching-data):
  // the `ignore` flag guards against setting state from a stale request
  // if the screen loses focus again before it resolves.
  useFocusEffect(
    useCallback(() => {
      let ignore = false;
      getTracks()
        .then((data) => {
          if (!ignore) {
            setTracks(data);
            setError(null);
          }
        })
        .catch((err) => {
          if (!ignore) {
            setError(err instanceof Error ? err.message : String(err));
          }
        });
      return () => {
        ignore = true;
      };
    }, [])
  );

  return (
    <SafeAreaView style={styles.container}>
      <Header title="Songs" action={{ label: 'Upload', onPress: () => router.push('/upload') }} />
      {error && (
        <View style={styles.errorBanner}>
          <Text style={styles.errorText}>{error}</Text>
        </View>
      )}
      <TrackList
        tracks={tracks}
        playingId={playingTrack?.id ?? null}
        onPress={(_, index) => playQueue(tracks, index)}
        onLongPress={setAddTarget}
      />
      <AddToPlaylistSheet track={addTarget} onClose={() => setAddTarget(null)} />
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
});
