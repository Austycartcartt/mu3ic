import { router, useFocusEffect } from 'expo-router';
import { useCallback, useState } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { getArtists, type Artist } from '@/api/client';
import { Header } from '@/components/Header';
import { SummaryList } from '@/components/SummaryList';
import { theme } from '@/theme/theme';

export default function ArtistsScreen() {
  const [artists, setArtists] = useState<Artist[]>([]);
  const [error, setError] = useState<string | null>(null);

  useFocusEffect(
    useCallback(() => {
      let ignore = false;
      getArtists()
        .then((data) => {
          if (!ignore) {
            setArtists(data);
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
      <Header title="Artists" />
      {error && (
        <View style={styles.errorBanner}>
          <Text style={styles.errorText}>{error}</Text>
        </View>
      )}
      <SummaryList
        items={artists}
        keyExtractor={(item) => item.artist}
        emptyMessage="No artists yet. Upload some tracks to get started."
        renderRow={(item) => (
          <Pressable
            style={styles.row}
            onPress={() => router.push(`/artists/${encodeURIComponent(item.artist)}`)}
          >
            <View style={styles.rowText}>
              <Text style={styles.title} numberOfLines={1}>
                {item.artist}
              </Text>
              <Text style={styles.subtitle}>
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
