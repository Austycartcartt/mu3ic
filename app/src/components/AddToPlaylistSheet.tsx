import { useCallback, useEffect, useState } from 'react';
import { Alert, FlatList, Modal, Pressable, StyleSheet, Text } from 'react-native';

import {
  addTrackToPlaylist,
  createPlaylist,
  getPlaylists,
  type Playlist,
  type Track,
} from '@/api/client';
import { theme } from '@/theme/theme';

import { PlaylistNameModal } from './PlaylistNameModal';

type Props = {
  // The track to add. Null keeps the sheet closed; setting it opens the
  // sheet (the parent screen owns this state).
  track: Track | null;
  onClose: () => void;
};

// Bottom sheet listing the user's playlists plus a "New playlist…" row.
// Opened by long-pressing a track row (see TrackList's onLongPress).
export function AddToPlaylistSheet({ track, onClose }: Props) {
  const [playlists, setPlaylists] = useState<Playlist[]>([]);
  const [creating, setCreating] = useState(false);
  const visible = track !== null;

  const load = useCallback(() => {
    getPlaylists()
      .then(setPlaylists)
      .catch((err) => Alert.alert('Could not load playlists', String(err)));
  }, []);

  useEffect(() => {
    if (visible) load();
  }, [visible, load]);

  async function addTo(playlist: Playlist) {
    if (!track) return;
    try {
      await addTrackToPlaylist(playlist.id, track.id);
      onClose();
      Alert.alert('Added', `“${track.title}” → ${playlist.name}`);
    } catch (err) {
      Alert.alert('Could not add track', String(err));
    }
  }

  async function createAndAdd(name: string) {
    if (!track) return;
    setCreating(false);
    try {
      const playlist = await createPlaylist(name);
      await addTrackToPlaylist(playlist.id, track.id);
      onClose();
      Alert.alert('Added', `“${track.title}” → ${playlist.name}`);
    } catch (err) {
      Alert.alert('Could not create playlist', String(err));
    }
  }

  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable style={styles.backdrop} onPress={onClose}>
        <Pressable style={styles.sheet} onPress={() => {}}>
          <Text style={styles.heading} numberOfLines={1}>
            Add “{track?.title}” to…
          </Text>

          <FlatList
            data={playlists}
            keyExtractor={(p) => String(p.id)}
            style={styles.list}
            renderItem={({ item }) => (
              <Pressable style={styles.row} onPress={() => addTo(item)}>
                <Text style={styles.rowText}>{item.name}</Text>
                <Text style={styles.rowMeta}>
                  {item.track_count} {item.track_count === 1 ? 'song' : 'songs'}
                </Text>
              </Pressable>
            )}
            ListEmptyComponent={<Text style={styles.empty}>No playlists yet.</Text>}
          />

          <Pressable style={styles.newRow} onPress={() => setCreating(true)}>
            <Text style={styles.newText}>+ New playlist…</Text>
          </Pressable>
        </Pressable>
      </Pressable>

      <PlaylistNameModal
        visible={creating}
        title="New playlist"
        submitLabel="Create"
        onSubmit={createAndAdd}
        onCancel={() => setCreating(false)}
      />
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.4)',
    justifyContent: 'flex-end',
  },
  sheet: {
    backgroundColor: theme.colors.background,
    borderTopLeftRadius: theme.radii.md,
    borderTopRightRadius: theme.radii.md,
    paddingHorizontal: theme.spacing.lg,
    paddingTop: theme.spacing.lg,
    paddingBottom: theme.spacing.xl,
    maxHeight: '70%',
  },
  heading: {
    fontSize: theme.fontSize.lg,
    fontWeight: 'bold',
    color: theme.colors.text,
    marginBottom: theme.spacing.md,
  },
  list: {
    flexGrow: 0,
  },
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: theme.spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.border,
  },
  rowText: {
    fontSize: theme.fontSize.lg,
    color: theme.colors.text,
  },
  rowMeta: {
    fontSize: theme.fontSize.sm,
    color: theme.colors.textMuted,
  },
  empty: {
    paddingVertical: theme.spacing.md,
    color: theme.colors.textMuted,
  },
  newRow: {
    paddingTop: theme.spacing.md,
  },
  newText: {
    fontSize: theme.fontSize.md,
    fontWeight: '600',
    color: theme.colors.accent,
  },
});
