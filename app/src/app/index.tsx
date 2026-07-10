import { useEffect, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { getTracks, streamUrl, type Track } from '@/api/client';
import { TrackList } from '@/components/TrackList';
import { usePlayer } from '@/hooks/usePlayer';

export default function TrackListScreen() {
  const [tracks, setTracks] = useState<Track[]>([]);
  const [error, setError] = useState<string | null>(null);
  const { play, playingId } = usePlayer();

  // Fetch-on-mount effect, following React's documented pattern
  // (https://react.dev/learn/synchronizing-with-effects#fetching-data):
  // the `ignore` flag guards against setting state from a stale request
  // if the component unmounts before it resolves.
  useEffect(() => {
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
  }, []);

  return (
    <SafeAreaView style={styles.container}>
      {error && (
        <View style={styles.errorBanner}>
          <Text style={styles.errorText}>{error}</Text>
        </View>
      )}
      <TrackList
        tracks={tracks}
        playingId={playingId}
        onPress={(track) => play(track.id, streamUrl(track.id))}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  errorBanner: {
    padding: 12,
    backgroundColor: '#fdd',
  },
  errorText: {
    color: '#900',
  },
});
