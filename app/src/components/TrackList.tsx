import { FlatList, Image, Pressable, StyleSheet, Text, View } from 'react-native';

import { artworkUrl, type Track } from '@/api/client';
import { theme } from '@/theme/theme';

type Props = {
  tracks: Track[];
  playingId: number | null;
  onPress: (track: Track) => void;
};

export function TrackList({ tracks, playingId, onPress }: Props) {
  return (
    <FlatList
      data={tracks}
      keyExtractor={(track) => String(track.id)}
      contentContainerStyle={styles.list}
      renderItem={({ item }) => {
        // "Unknown" is a real, always-present value now (server default),
        // not an empty string — filter it out explicitly rather than
        // relying on falsiness, or it'd render as a literal "Unknown".
        const subtitle = [item.artist, item.album]
          .filter((v) => v !== 'Unknown')
          .join(' — ');
        return (
          <Pressable style={styles.row} onPress={() => onPress(item)}>
            {item.hasArtwork ? (
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
            {playingId === item.id && <Text style={styles.playing}>▶</Text>}
          </Pressable>
        );
      }}
      ListEmptyComponent={<Text style={styles.empty}>No tracks yet. Tap Upload to add some.</Text>}
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
