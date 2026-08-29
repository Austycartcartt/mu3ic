import { useEffect, useRef, useState } from 'react';
import { StyleSheet, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { searchTracks, type Track } from '@/api/client';
import { AddToPlaylistSheet } from '@/components/AddToPlaylistSheet';
import { Header } from '@/components/Header';
import { TrackList } from '@/components/TrackList';
import { usePlayer } from '@/hooks/usePlayer';
import { theme } from '@/theme/theme';

export default function SearchScreen() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<Track[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [addTarget, setAddTarget] = useState<Track | null>(null);
  const { playQueue, playingTrack } = usePlayer();

  const trimmed = query.trim();

  // Debounce the query ~300ms, and guard against an earlier request
  // resolving after a later one (same `ignore` pattern as the list
  // screens' useFocusEffect).
  useEffect(() => {
    if (trimmed === '') {
      setResults([]);
      setError(null);
      return;
    }
    let ignore = false;
    const handle = setTimeout(() => {
      searchTracks(trimmed)
        .then((data) => {
          if (!ignore) {
            setResults(data);
            setError(null);
          }
        })
        .catch((err) => {
          if (!ignore) setError(err instanceof Error ? err.message : String(err));
        });
    }, 300);
    return () => {
      ignore = true;
      clearTimeout(handle);
    };
  }, [trimmed]);

  const inputRef = useRef<TextInput>(null);

  return (
    <SafeAreaView style={styles.container}>
      <Header title="Search" />
      <View style={styles.searchBar}>
        <TextInput
          ref={inputRef}
          style={styles.input}
          placeholder="Songs, artists, albums"
          value={query}
          onChangeText={setQuery}
          autoCapitalize="none"
          autoCorrect={false}
          returnKeyType="search"
          clearButtonMode="while-editing"
        />
      </View>
      {error && (
        <View style={styles.errorBanner}>
          <Text style={styles.errorText}>{error}</Text>
        </View>
      )}
      {trimmed === '' ? (
        <View style={styles.hint}>
          <Text style={styles.hintText}>Search your library</Text>
        </View>
      ) : (
        <TrackList
          tracks={results}
          playingId={playingTrack?.id ?? null}
          onPress={(_, index) => playQueue(results, index)}
          onLongPress={setAddTarget}
          emptyMessage={`No matches for “${trimmed}”`}
        />
      )}
      <AddToPlaylistSheet track={addTarget} onClose={() => setAddTarget(null)} />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  searchBar: {
    paddingHorizontal: theme.spacing.lg,
    paddingVertical: theme.spacing.sm,
  },
  input: {
    borderWidth: 1,
    borderColor: theme.colors.border,
    borderRadius: theme.radii.sm,
    paddingVertical: theme.spacing.sm,
    paddingHorizontal: theme.spacing.md,
    fontSize: theme.fontSize.md,
  },
  errorBanner: {
    padding: theme.spacing.md,
    backgroundColor: theme.colors.dangerBackground,
  },
  errorText: {
    color: theme.colors.danger,
  },
  hint: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  hintText: {
    fontSize: theme.fontSize.lg,
    color: theme.colors.textMuted,
  },
});
