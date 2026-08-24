import { FlatList, StyleSheet, Text } from 'react-native';

import { theme } from '@/theme/theme';

type Props<T> = {
  items: T[];
  keyExtractor: (item: T) => string;
  renderRow: (item: T) => React.ReactElement;
  emptyMessage: string;
};

// Generic row-list chrome (padding, separators, empty state) shared by the
// Artists and Albums top-level screens — same FlatList shape as TrackList,
// but TrackList is Track[]-typed so it isn't a fit for artist/album summary
// rows. renderRow is expected to return a Pressable already wired to the
// caller's own onPress handler.
export function SummaryList<T>({ items, keyExtractor, renderRow, emptyMessage }: Props<T>) {
  return (
    <FlatList
      data={items}
      keyExtractor={keyExtractor}
      contentContainerStyle={styles.list}
      renderItem={({ item }) => renderRow(item)}
      ListEmptyComponent={<Text style={styles.empty}>{emptyMessage}</Text>}
    />
  );
}

const styles = StyleSheet.create({
  list: {
    padding: theme.spacing.lg,
  },
  empty: {
    marginTop: theme.spacing.xl,
    textAlign: 'center',
    color: theme.colors.textMuted,
  },
});
