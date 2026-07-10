import { FlatList, Pressable, StyleSheet, Text, View } from 'react-native';

import type { Track } from '@/api/client';

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
      renderItem={({ item }) => (
        <Pressable style={styles.row} onPress={() => onPress(item)}>
          <View style={styles.rowText}>
            <Text style={styles.filename}>{item.filename}</Text>
          </View>
          {playingId === item.id && <Text style={styles.playing}>▶</Text>}
        </Pressable>
      )}
      ListEmptyComponent={<Text style={styles.empty}>No tracks yet. Upload one via curl — see server/README.md.</Text>}
    />
  );
}

const styles = StyleSheet.create({
  list: {
    padding: 16,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 12,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#ccc',
  },
  rowText: {
    flex: 1,
  },
  filename: {
    fontSize: 16,
  },
  playing: {
    fontSize: 16,
  },
  empty: {
    marginTop: 32,
    textAlign: 'center',
    color: '#666',
  },
});
