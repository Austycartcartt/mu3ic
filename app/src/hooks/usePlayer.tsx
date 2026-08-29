import { useAudioPlayer, useAudioPlayerStatus } from 'expo-audio';
import { createContext, useContext, useEffect, useRef, useState, type ReactNode } from 'react';

import { streamUrl, type Track } from '@/api/client';

// Wraps expo-audio's player hook (never expo-av — it's deprecated) as a
// context so playback state (and the docked player UI) survives navigating
// between screens instead of resetting per-screen.
//
// Playback is queue-based: callers hand over the whole list they're
// showing (an album, a playlist, search results) plus the tapped index,
// and the player advances through it — on the ⏭/⏮ dock buttons and
// automatically when a track finishes.
type PlayerContextValue = {
  playQueue: (tracks: Track[], startIndex: number) => void;
  togglePlayPause: () => void;
  seek: (seconds: number) => void;
  playNext: () => void;
  playPrevious: () => void;
  playingTrack: Track | null;
  isPlaying: boolean;
  hasNext: boolean;
  hasPrevious: boolean;
  currentTime: number;
  duration: number;
  isBuffering: boolean;
};

const PlayerContext = createContext<PlayerContextValue | null>(null);

export function PlayerProvider({ children }: { children: ReactNode }) {
  const player = useAudioPlayer();
  const status = useAudioPlayerStatus(player);
  const [queue, setQueue] = useState<Track[]>([]);
  const [index, setIndex] = useState(0);

  const playingTrack = queue[index] ?? null;
  const hasNext = index < queue.length - 1;
  const hasPrevious = index > 0;

  function playAt(tracks: Track[], at: number) {
    const track = tracks[at];
    if (!track) return;
    player.replace({ uri: streamUrl(track.id) });
    player.play();
  }

  function playQueue(tracks: Track[], startIndex: number) {
    setQueue(tracks);
    setIndex(startIndex);
    playAt(tracks, startIndex);
  }

  function playNext() {
    if (!hasNext) return;
    const next = index + 1;
    setIndex(next);
    playAt(queue, next);
  }

  function playPrevious() {
    if (!hasPrevious) return;
    const prev = index - 1;
    setIndex(prev);
    playAt(queue, prev);
  }

  function togglePlayPause() {
    if (status.playing) {
      player.pause();
    } else {
      player.play();
    }
  }

  function seek(seconds: number) {
    if (!Number.isFinite(seconds)) return;
    player.seekTo(seconds);
  }

  // Auto-advance when a track finishes. status.didJustFinish stays true
  // across a few status frames, so a ref gates it to one advance per
  // finish. When there's no next track, playback just stops.
  const advancedForFinish = useRef(false);
  useEffect(() => {
    if (!status.didJustFinish) {
      advancedForFinish.current = false;
      return;
    }
    if (advancedForFinish.current) return;
    advancedForFinish.current = true;
    if (hasNext) {
      const next = index + 1;
      setIndex(next);
      playAt(queue, next);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status.didJustFinish]);

  const value: PlayerContextValue = {
    playQueue,
    togglePlayPause,
    seek,
    playNext,
    playPrevious,
    playingTrack,
    isPlaying: status.playing,
    hasNext,
    hasPrevious,
    currentTime: status.currentTime,
    duration: status.duration || playingTrack?.duration_seconds || 0,
    isBuffering: status.isBuffering,
  };

  return <PlayerContext.Provider value={value}>{children}</PlayerContext.Provider>;
}

export function usePlayer() {
  const context = useContext(PlayerContext);
  if (!context) {
    throw new Error('usePlayer must be used within a PlayerProvider');
  }
  return context;
}
