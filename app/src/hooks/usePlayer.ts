import { useAudioPlayer, useAudioPlayerStatus } from 'expo-audio';
import { useState } from 'react';

// Wraps expo-audio's player hook (never expo-av — it's deprecated) and
// tracks which track id is currently playing, so the track list can show
// a "now playing" indicator.
export function usePlayer() {
  const player = useAudioPlayer();
  const status = useAudioPlayerStatus(player);
  const [playingId, setPlayingId] = useState<number | null>(null);

  function play(id: number, uri: string) {
    player.replace({ uri });
    player.play();
    setPlayingId(id);
  }

  return { play, playingId, isPlaying: status.playing };
}
