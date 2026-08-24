import { router, useFocusEffect } from 'expo-router';
import { useCallback, useState } from 'react';
import { Image, Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { artworkUrl, getAlbums, type Album } from '@/api/client';
import { Header } from '@/components/Header';
import { SummaryList } from '@/components/SummaryList';
import { theme } from '@/theme/theme';

export default function AlbumsScreen() {
  const [albums, setAlbums] = useState<Album[]>([]);
  const [error, setError] = useState<string | null>(null);

  useFocusEffect(
    useCallback(() => {
      let ignore = false;
      getAlbums()
        .then((data) => {
          if (!ignore) {
            setAlbums(data);
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
      <Header title="Albums" />
      {error && (
        <View style={styles.errorBanner}>
          <Text style={styles.errorText}>{error}</Text>
        </View>
      )}
      <SummaryList
        items={albums}
        keyExtractor={(item) => `${item.album}—${item.artist}`}
        emptyMessage="No albums yet. Upload some tracks to get started."
        renderRow={(item) => (
          <Pressable
            style={styles.row}
            onPress={() =>
              router.push({
                pathname: '/albums/[name]',
                params: { name: item.album, artist: item.artist },
              })
            }
          >
            {item.hasArtwork ? (
              <Image source={{ uri: artworkUrl(item.representative_track_id) }} style={styles.artwork} />
            ) : (
              <View style={[styles.artwork, styles.artworkPlaceholder]} />
            )}
            <View style={styles.rowText}>
              <Text style={styles.title} numberOfLines={1}>
                {item.album}
              </Text>
              <Text style={styles.subtitle} numberOfLines={1}>
                {item.artist !== 'Unknown' ? item.artist : ''}
                {item.artist !== 'Unknown' ? ' — ' : ''}
                {item.track_count} {item.track_count === 1 ? 'song' : 'songs'}
              </Text>
            </View>
          </Pressable>
        )}
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
});
