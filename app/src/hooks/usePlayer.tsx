import { useAudioPlayer, useAudioPlayerStatus } from 'expo-audio';
import { createContext, useContext, useState, type ReactNode } from 'react';

import type { Track } from '@/api/client';

// Wraps expo-audio's player hook (never expo-av — it's deprecated) as a
// context so playback state (and the docked player UI) survives navigating
// between screens instead of resetting per-screen.
type PlayerContextValue = {
  play: (track: Track, uri: string) => void;
  togglePlayPause: () => void;
  seek: (seconds: number) => void;
  playingTrack: Track | null;
  isPlaying: boolean;
  currentTime: number;
  duration: number;
  isBuffering: boolean;
};

const PlayerContext = createContext<PlayerContextValue | null>(null);

export function PlayerProvider({ children }: { children: ReactNode }) {
  const player = useAudioPlayer();
  const status = useAudioPlayerStatus(player);
  const [playingTrack, setPlayingTrack] = useState<Track | null>(null);

  function play(track: Track, uri: string) {
    player.replace({ uri });
    player.play();
    setPlayingTrack(track);
  }

  function togglePlayPause() {
    if (status.playing) {
      player.pause();
    } else {
      player.play();
    }
  }

  function seek(seconds: number) {
    player.seekTo(seconds);
  }

  const value: PlayerContextValue = {
    play,
    togglePlayPause,
    seek,
    playingTrack,
    isPlaying: status.playing,
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
