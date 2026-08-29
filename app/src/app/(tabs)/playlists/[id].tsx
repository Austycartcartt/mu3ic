import { router, useFocusEffect, useLocalSearchParams } from 'expo-router';
import { useCallback, useState } from 'react';
import { Alert, Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import {
  deletePlaylist,
  getPlaylistTracks,
  removeTrackFromPlaylist,
  renamePlaylist,
  reorderPlaylist,
  type Track,
} from '@/api/client';
import { Header } from '@/components/Header';
import { PlaylistNameModal } from '@/components/PlaylistNameModal';
import { TrackList } from '@/components/TrackList';
import { usePlayer } from '@/hooks/usePlayer';
import { theme } from '@/theme/theme';

export default function PlaylistDetailScreen() {
  const params = useLocalSearchParams<{ id: string; name?: string }>();
  const id = Number(params.id);
  const [displayName, setDisplayName] = useState(params.name ?? 'Playlist');
  const [tracks, setTracks] = useState<Track[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState(false);
  const [renaming, setRenaming] = useState(false);
  const { playQueue, playingTrack } = usePlayer();

  const load = useCallback(() => {
    let ignore = false;
    getPlaylistTracks(id)
      .then((data) => {
        if (!ignore) {
          setTracks(data);
          setError(null);
        }
      })
      .catch((err) => {
        if (!ignore) setError(err instanceof Error ? err.message : String(err));
      });
    return () => {
      ignore = true;
    };
  }, [id]);

  useFocusEffect(load);

  // Reorder is optimistic: swap locally, push the new order, and reload
  // from the server on failure so the UI can't drift from the DB.
  async function move(from: number, to: number) {
    if (to < 0 || to >= tracks.length) return;
    const next = [...tracks];
    [next[from], next[to]] = [next[to], next[from]];
    setTracks(next);
    try {
      await reorderPlaylist(id, next.map((t) => t.id));
    } catch (err) {
      Alert.alert('Could not reorder', String(err));
      load();
    }
  }

  async function remove(track: Track) {
    const next = tracks.filter((t) => t.id !== track.id);
    setTracks(next);
    try {
      await removeTrackFromPlaylist(id, track.id);
    } catch (err) {
      Alert.alert('Could not remove track', String(err));
      load();
    }
  }

  async function handleRename(name: string) {
    setRenaming(false);
    try {
      await renamePlaylist(id, name);
      setDisplayName(name);
    } catch (err) {
      Alert.alert('Could not rename', String(err));
    }
  }

  function confirmDelete() {
    Alert.alert('Delete playlist?', displayName, [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Delete',
        style: 'destructive',
        onPress: async () => {
          try {
            await deletePlaylist(id);
            router.back();
          } catch (err) {
            Alert.alert('Could not delete', String(err));
          }
        },
      },
    ]);
  }

  return (
    <SafeAreaView style={styles.container}>
      <Header
        title={displayName}
        action={{ label: editing ? 'Done' : 'Edit', onPress: () => setEditing((v) => !v) }}
      />
      {editing && (
        <View style={styles.editBar}>
          <Pressable onPress={() => setRenaming(true)}>
            <Text style={styles.editAction}>Rename</Text>
          </Pressable>
          <Pressable onPress={confirmDelete}>
            <Text style={[styles.editAction, styles.danger]}>Delete playlist</Text>
          </Pressable>
        </View>
      )}
      {error && (
        <View style={styles.errorBanner}>
          <Text style={styles.errorText}>{error}</Text>
        </View>
      )}
      <TrackList
        tracks={tracks}
        playingId={playingTrack?.id ?? null}
        onPress={(_, index) => playQueue(tracks, index)}
        emptyMessage="This playlist is empty. Long-press a song anywhere to add it."
        editControls={
          editing
            ? {
                onMoveUp: (index) => move(index, index - 1),
                onMoveDown: (index) => move(index, index + 1),
                onRemove: remove,
              }
            : undefined
        }
      />
      <PlaylistNameModal
        visible={renaming}
        title="Rename playlist"
        initialValue={displayName}
        onSubmit={handleRename}
        onCancel={() => setRenaming(false)}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  editBar: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing.lg,
    paddingVertical: theme.spacing.sm,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.border,
    backgroundColor: theme.colors.surface,
  },
  editAction: {
    fontSize: theme.fontSize.md,
    fontWeight: '600',
    color: theme.colors.accent,
  },
  danger: {
    color: theme.colors.danger,
  },
  errorBanner: {
    padding: theme.spacing.md,
    backgroundColor: theme.colors.dangerBackground,
  },
  errorText: {
    color: theme.colors.danger,
  },
});
