import { useState } from 'react';
import { Pressable, StyleSheet, Text, View, type LayoutChangeEvent, type GestureResponderEvent } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { usePlayer } from '@/hooks/usePlayer';
import { theme } from '@/theme/theme';

function formatTime(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds));
  const minutes = Math.floor(total / 60);
  const secs = total % 60;
  return `${minutes}:${secs.toString().padStart(2, '0')}`;
}

// Fixed dock, not a scroll-flow card — stays mounted at the bottom of the
// app (see _layout.tsx) so it persists across screen navigation.
export function PlayerDock() {
  const { playingTrack, isPlaying, currentTime, duration, togglePlayPause, seek } = usePlayer();
  const insets = useSafeAreaInsets();
  const [trackWidth, setTrackWidth] = useState(0);

  if (!playingTrack) {
    return null;
  }

  const progress = duration > 0 ? currentTime / duration : 0;

  function handleLayout(event: LayoutChangeEvent) {
    setTrackWidth(event.nativeEvent.layout.width);
  }

  function handleSeekPress(event: GestureResponderEvent) {
    if (trackWidth === 0 || duration === 0) return;
    const ratio = Math.min(1, Math.max(0, event.nativeEvent.locationX / trackWidth));
    seek(ratio * duration);
  }

  const subtitle = playingTrack.artist !== 'Unknown' ? playingTrack.artist : '';

  return (
    <View style={[styles.container, { paddingBottom: insets.bottom + theme.spacing.sm }]}>
      <View style={styles.info}>
        <Text style={styles.title} numberOfLines={1}>
          {playingTrack.title}
        </Text>
        {subtitle !== '' && (
          <Text style={styles.subtitle} numberOfLines={1}>
            {subtitle}
          </Text>
        )}
      </View>

      <Pressable style={styles.progressTrack} onLayout={handleLayout} onPress={handleSeekPress}>
        <View style={styles.progressBase} />
        <View style={[styles.progressFill, { width: `${progress * 100}%` }]} />
      </Pressable>

      <View style={styles.row}>
        <Text style={styles.time}>{formatTime(currentTime)}</Text>
        <Pressable style={styles.playButton} onPress={togglePlayPause}>
          <Text style={styles.playButtonText}>{isPlaying ? '⏸' : '▶'}</Text>
        </Pressable>
        <Text style={styles.time}>-{formatTime(duration - currentTime)}</Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    paddingHorizontal: theme.spacing.lg,
    paddingTop: theme.spacing.sm,
    backgroundColor: theme.colors.surface,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.border,
  },
  info: {
    marginBottom: theme.spacing.xs,
  },
  title: {
    fontSize: theme.fontSize.md,
    fontWeight: '600',
    color: theme.colors.text,
  },
  subtitle: {
    fontSize: theme.fontSize.sm,
    color: theme.colors.textMuted,
  },
  progressTrack: {
    height: 20,
    justifyContent: 'center',
  },
  progressBase: {
    position: 'absolute',
    left: 0,
    right: 0,
    height: 4,
    borderRadius: theme.radii.sm,
    backgroundColor: theme.colors.border,
  },
  progressFill: {
    position: 'absolute',
    left: 0,
    height: 4,
    borderRadius: theme.radii.sm,
    backgroundColor: theme.colors.accent,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  time: {
    fontSize: theme.fontSize.sm,
    color: theme.colors.textMuted,
    width: 44,
  },
  playButton: {
    paddingHorizontal: theme.spacing.lg,
    paddingVertical: theme.spacing.xs,
  },
  playButtonText: {
    fontSize: theme.fontSize.xl,
    color: theme.colors.text,
  },
});
