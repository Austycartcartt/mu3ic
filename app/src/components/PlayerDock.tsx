import { useRef, useState } from 'react';
import { Pressable, StyleSheet, Text, View, type GestureResponderEvent } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { usePlayer } from '@/hooks/usePlayer';
import { theme } from '@/theme/theme';

function formatTime(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds));
  const minutes = Math.floor(total / 60);
  const secs = total % 60;
  return `${minutes}:${secs.toString().padStart(2, '0')}`;
}

type Props = {
  // False when something below this dock already accounts for the bottom
  // safe area (e.g. the tab bar in (tabs)/_layout.tsx) — otherwise the
  // inset would be applied twice.
  withSafeAreaInset?: boolean;
};

// Fixed dock, not a scroll-flow card — stays mounted persistently so it
// survives screen navigation (see (tabs)/_layout.tsx and upload.tsx).
export function PlayerDock({ withSafeAreaInset = true }: Props) {
  const {
    playingTrack,
    isPlaying,
    currentTime,
    duration,
    togglePlayPause,
    seek,
    playNext,
    playPrevious,
    hasNext,
    hasPrevious,
  } = usePlayer();
  const insets = useSafeAreaInsets();
  const [trackWidth, setTrackWidth] = useState(0);
  const trackPageX = useRef(0);
  const trackRef = useRef<View>(null);

  if (!playingTrack) {
    return null;
  }

  const bottomInset = withSafeAreaInset ? insets.bottom : 0;
  const progress = duration > 0 ? currentTime / duration : 0;

  // nativeEvent.locationX is unreliable on Fabric (RN 0.81) — it's not
  // consistently relative to the pressed element — so measure the track's
  // page position on layout and derive the seek ratio from pageX instead.
  function handleLayout() {
    trackRef.current?.measure((_x, _y, width, _height, pageX) => {
      setTrackWidth(width);
      trackPageX.current = pageX;
    });
  }

  function handleSeekPress(event: GestureResponderEvent) {
    if (trackWidth === 0 || !Number.isFinite(duration) || duration <= 0) return;
    const ratio = Math.min(1, Math.max(0, (event.nativeEvent.pageX - trackPageX.current) / trackWidth));
    seek(ratio * duration);
  }

  const subtitle = playingTrack.artist !== 'Unknown' ? playingTrack.artist : '';

  return (
    <View style={[styles.container, { paddingBottom: bottomInset + theme.spacing.sm }]}>
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

      <Pressable ref={trackRef} style={styles.progressTrack} onLayout={handleLayout} onPress={handleSeekPress}>
        <View style={styles.progressBase} />
        <View style={[styles.progressFill, { width: `${progress * 100}%` }]} />
      </Pressable>

      <View style={styles.row}>
        <Text style={styles.time}>{formatTime(currentTime)}</Text>
        <View style={styles.controls}>
          <Pressable onPress={playPrevious} disabled={!hasPrevious} hitSlop={8}>
            <Text style={[styles.skip, !hasPrevious && styles.skipDisabled]}>⏮</Text>
          </Pressable>
          <Pressable style={styles.playButton} onPress={togglePlayPause}>
            <Text style={styles.playButtonText}>{isPlaying ? '⏸' : '▶'}</Text>
          </Pressable>
          <Pressable onPress={playNext} disabled={!hasNext} hitSlop={8}>
            <Text style={[styles.skip, !hasNext && styles.skipDisabled]}>⏭</Text>
          </Pressable>
        </View>
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
  controls: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing.md,
  },
  time: {
    fontSize: theme.fontSize.sm,
    color: theme.colors.textMuted,
    width: 44,
  },
  skip: {
    fontSize: theme.fontSize.lg,
    color: theme.colors.text,
  },
  skipDisabled: {
    color: theme.colors.border,
  },
  playButton: {
    paddingHorizontal: theme.spacing.md,
    paddingVertical: theme.spacing.xs,
  },
  playButtonText: {
    fontSize: theme.fontSize.xl,
    color: theme.colors.text,
  },
});
