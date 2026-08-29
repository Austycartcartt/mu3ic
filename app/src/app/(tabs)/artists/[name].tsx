import { useLocalSearchParams, useFocusEffect } from 'expo-router';
import { useCallback, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { getArtistTracks, type Track } from '@/api/client';
import { AddToPlaylistSheet } from '@/components/AddToPlaylistSheet';
import { Header } from '@/components/Header';
import { TrackList } from '@/components/TrackList';
import { usePlayer } from '@/hooks/usePlayer';
import { theme } from '@/theme/theme';

export default function ArtistTracksScreen() {
  const { name } = useLocalSearchParams<{ name: string }>();
  const [tracks, setTracks] = useState<Track[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [addTarget, setAddTarget] = useState<Track | null>(null);
  const { playQueue, playingTrack } = usePlayer();

  useFocusEffect(
    useCallback(() => {
      let ignore = false;
      getArtistTracks(name)
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
    }, [name])
  );

  return (
    <SafeAreaView style={styles.container}>
      <Header title={name} />
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
