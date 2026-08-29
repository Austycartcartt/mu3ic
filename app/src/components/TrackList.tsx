import { FlatList, Image, Pressable, StyleSheet, Text, View } from 'react-native';

import { artworkUrl, type Track } from '@/api/client';
import { theme } from '@/theme/theme';

// Reorder + remove controls for the playlist screen's edit mode. When
// present, rows swap their artwork for ▲/▼ buttons and their playing
// indicator for a ✕ — see app/src/app/(tabs)/playlists/[id].tsx.
type EditControls = {
  onMoveUp: (index: number) => void;
  onMoveDown: (index: number) => void;
  onRemove: (track: Track) => void;
};

type Props = {
  tracks: Track[];
  playingId: number | null;
  onPress: (track: Track, index: number) => void;
  // Long-press a row to add it to a playlist (Songs / album / artist /
  // search screens). Omitted where it doesn't apply.
  onLongPress?: (track: Track) => void;
  // Shows each track's position number instead of artwork — only makes
  // sense for a single-album list, where position is meaningful; the
  // all-tracks and per-artist lists mix albums so it's omitted there.
  showTrackNumbers?: boolean;
  editControls?: EditControls;
  emptyMessage?: string;
};

export function TrackList({
  tracks,
  playingId,
  onPress,
  onLongPress,
  showTrackNumbers,
  editControls,
  emptyMessage,
}: Props) {
  return (
    <FlatList
      data={tracks}
      keyExtractor={(track) => String(track.id)}
      contentContainerStyle={styles.list}
      renderItem={({ item, index }) => {
        // "Unknown" is a real, always-present value now (server default),
        // not an empty string — filter it out explicitly rather than
        // relying on falsiness, or it'd render as a literal "Unknown".
        const subtitle = [item.artist, item.album]
          .filter((v) => v !== 'Unknown')
          .join(' — ');
        return (
          <Pressable
            style={styles.row}
            onPress={() => onPress(item, index)}
            onLongPress={onLongPress ? () => onLongPress(item) : undefined}
          >
            {editControls ? (
              <View style={styles.reorder}>
                <Pressable
                  onPress={() => editControls.onMoveUp(index)}
                  disabled={index === 0}
                  hitSlop={8}
                >
                  <Text style={[styles.reorderBtn, index === 0 && styles.reorderBtnDisabled]}>▲</Text>
                </Pressable>
                <Pressable
                  onPress={() => editControls.onMoveDown(index)}
                  disabled={index === tracks.length - 1}
                  hitSlop={8}
                >
                  <Text
                    style={[
                      styles.reorderBtn,
                      index === tracks.length - 1 && styles.reorderBtnDisabled,
                    ]}
                  >
                    ▼
                  </Text>
                </Pressable>
              </View>
            ) : showTrackNumbers ? (
              <Text style={styles.trackNumber}>{item.track_number ?? ''}</Text>
            ) : item.hasArtwork ? (
              // React Native's built-in Image caching is enough here — no
              // image-caching library per the Phase 2 spec.
              <Image source={{ uri: artworkUrl(item.id) }} style={styles.artwork} />
            ) : (
              <View style={[styles.artwork, styles.artworkPlaceholder]} />
            )}
            <View style={styles.rowText}>
              <Text style={styles.title} numberOfLines={1}>
                {item.title}
              </Text>
              {subtitle !== '' && (
                <Text style={styles.subtitle} numberOfLines={1}>
                  {subtitle}
                </Text>
              )}
            </View>
            {editControls ? (
              <Pressable onPress={() => editControls.onRemove(item)} hitSlop={8}>
                <Text style={styles.remove}>✕</Text>
              </Pressable>
            ) : (
              playingId === item.id && <Text style={styles.playing}>▶</Text>
            )}
          </Pressable>
        );
      }}
      ListEmptyComponent={
        <Text style={styles.empty}>
          {emptyMessage ?? 'No tracks yet. Tap Upload to add some.'}
        </Text>
      }
    />
  );
}

const styles = StyleSheet.create({
  list: {
    padding: theme.spacing.lg,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: theme.spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.border,
  },
  trackNumber: {
    width: 24,
    textAlign: 'right',
    marginRight: theme.spacing.md,
    color: theme.colors.textMuted,
    fontSize: theme.fontSize.sm,
  },
  reorder: {
    width: 44,
    marginRight: theme.spacing.md,
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  reorderBtn: {
    fontSize: theme.fontSize.lg,
    color: theme.colors.accent,
  },
  reorderBtnDisabled: {
    color: theme.colors.border,
  },
  remove: {
    fontSize: theme.fontSize.lg,
    color: theme.colors.danger,
    paddingHorizontal: theme.spacing.sm,
  },
  artwork: {
    width: 44,
    height: 44,
    borderRadius: theme.radii.sm,
    marginRight: theme.spacing.md,
  },
  artworkPlaceholder: {
    backgroundColor: theme.colors.surface,
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
  playing: {
    fontSize: theme.fontSize.lg,
  },
  empty: {
    marginTop: theme.spacing.xl,
    textAlign: 'center',
    color: theme.colors.textMuted,
  },
});
