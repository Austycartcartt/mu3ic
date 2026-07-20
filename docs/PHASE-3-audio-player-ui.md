# Phase 3: Audio Player UI

**Status:** Planned

## Goal

Enhance the bare-bones play button into a full-featured audio player UI. Users should be able to see track progress, remaining time, pause playback, and seek through the track.

---

## Current State

Right now you have:
- A play button that starts streaming
- `expo-audio` handles playback
- That's it

**What's missing:**
- Pause button
- Play/pause toggle UI
- Progress bar (slider showing current position)
- Current time display (e.g., "1:23")
- Duration display (e.g., "4:56")
- Time remaining (e.g., "-3:33")
- Seeking (tap on progress bar to jump)
- Visual feedback (playing vs. paused state)

---

## Why This Matters Before Phase 5 (Auth)

A proper player is table-stakes UX. Users expect:
1. To pause mid-track (not just stop/play from start)
2. To see progress (how far through the track)
3. To seek forward/backward
4. To know how long the track is

Without this, the app feels broken, even if auth works.

---

## Implementation Plan

### 1. Extend Database Schema

Currently, tracks have `title`, `artist`, `album`, `mime_type`, `original_filename`, `created_at`.

Add a computed field for duration (populate during metadata extraction in Phase 2, but if not available, set during initial streaming):

```sql
ALTER TABLE tracks ADD COLUMN duration_ms INTEGER; -- duration in milliseconds
```

When extracting metadata via `dhowden/tag`, capture:

```go
metadata := tag.Read(file)
durationMs := int(metadata.Duration().Milliseconds())
// Store in DB
```

### 2. API Endpoint Update

**GET /api/tracks** should now return duration:

```json
[
  {
    "id": "uuid",
    "title": "Track Name",
    "artist": "Artist",
    "album": "Album",
    "duration_ms": 256000,  // 4 minutes 16 seconds
    "mime_type": "audio/mpeg"
  }
]
```

### 3. Audio Player Component

Replace your current bare play button with a full player:

```typescript
// components/AudioPlayer.tsx
import { useState, useEffect, useRef } from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { Audio } from 'expo-audio';
import Slider from '@react-native-community/slider'; // Need to install

type Track = {
  id: string;
  title: string;
  artist: string;
  duration_ms: number;
};

type AudioPlayerProps = {
  track: Track;
  streamUrl: string;
};

export function AudioPlayer({ track, streamUrl }: AudioPlayerProps) {
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentPositionMs, setCurrentPositionMs] = useState(0);
  const [duration, setDuration] = useState(track.duration_ms);
  const soundRef = useRef<Audio.Sound | null>(null);

  // Initialize audio session and load sound
  useEffect(() => {
    async function setupAudio() {
      try {
        // Set audio mode for playback (not recording)
        await Audio.setAudioModeAsync({
          allowsRecordingIOS: false,
          interruptionModeIOS: Audio.INTERRUPTION_MODE_IOS_DO_NOT_MIX,
          interruptionModeAndroid: Audio.INTERRUPTION_MODE_ANDROID_DO_NOT_MIX,
          shouldDuckAndroid: true,
          playThroughEarpieceAndroid: false,
        });

        // Load sound from stream URL
        const { sound } = await Audio.Sound.createAsync(
          { uri: streamUrl },
          { shouldPlay: false }
        );

        soundRef.current = sound;

        // Set duration from loaded sound (fallback if DB doesn't have it)
        const status = await sound.getStatusAsync();
        if (status.isLoaded && status.durationMillis) {
          setDuration(status.durationMillis);
        }

        // Subscribe to playback status updates
        sound.setOnPlaybackStatusUpdate(handleStatusUpdate);
      } catch (error) {
        console.error('Failed to load audio', error);
      }
    }

    setupAudio();

    return () => {
      if (soundRef.current) {
        soundRef.current.unloadAsync();
      }
    };
  }, [streamUrl]);

  const handleStatusUpdate = (status: Audio.PlaybackStatus) => {
    if (status.isLoaded) {
      setCurrentPositionMs(status.positionMillis);
      // Stop when reaching end
      if (status.didJustFinish && !status.isLooping) {
        setIsPlaying(false);
      }
    }
  };

  const togglePlayPause = async () => {
    if (!soundRef.current) return;

    try {
      if (isPlaying) {
        await soundRef.current.pauseAsync();
        setIsPlaying(false);
      } else {
        await soundRef.current.playAsync();
        setIsPlaying(true);
      }
    } catch (error) {
      console.error('Playback control failed', error);
    }
  };

  const handleSeek = async (value: number) => {
    if (!soundRef.current) return;

    try {
      await soundRef.current.setPositionAsync(value);
      setCurrentPositionMs(value);
    } catch (error) {
      console.error('Seek failed', error);
    }
  };

  // Format milliseconds to MM:SS
  const formatTime = (ms: number): string => {
    const totalSeconds = Math.floor(ms / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  };

  const timeRemaining = duration - currentPositionMs;
  const isBuffering = false; // Simplified; could track buffering state

  return (
    <View style={styles.container}>
      {/* Track Info */}
      <View style={styles.trackInfo}>
        <Text style={styles.title}>{track.title}</Text>
        <Text style={styles.artist}>{track.artist}</Text>
      </View>

      {/* Progress Bar */}
      <View style={styles.progressContainer}>
        <Slider
          style={styles.slider}
          minimumValue={0}
          maximumValue={duration}
          value={currentPositionMs}
          onSlidingComplete={handleSeek}
          minimumTrackTintColor="#007AFF"
          maximumTrackTintColor="#E5E5EA"
          thumbTintColor="#007AFF"
        />
      </View>

      {/* Time Display */}
      <View style={styles.timeContainer}>
        <Text style={styles.time}>{formatTime(currentPositionMs)}</Text>
        <Text style={styles.time}>-{formatTime(timeRemaining)}</Text>
      </View>

      {/* Controls */}
      <View style={styles.controls}>
        <TouchableOpacity
          onPress={togglePlayPause}
          style={[styles.button, isPlaying && styles.buttonActive]}
        >
          <Text style={styles.buttonText}>
            {isPlaying ? '⏸ Pause' : '▶ Play'}
          </Text>
        </TouchableOpacity>
      </View>

      {isBuffering && <Text style={styles.buffering}>Loading...</Text>}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    paddingHorizontal: 16,
    paddingVertical: 20,
    backgroundColor: '#F9F9F9',
    borderRadius: 12,
    marginBottom: 20,
  },
  trackInfo: {
    marginBottom: 16,
  },
  title: {
    fontSize: 18,
    fontWeight: 'bold',
    marginBottom: 4,
  },
  artist: {
    fontSize: 14,
    color: '#666',
  },
  progressContainer: {
    marginVertical: 12,
  },
  slider: {
    width: '100%',
    height: 40,
  },
  timeContainer: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 16,
  },
  time: {
    fontSize: 12,
    color: '#666',
  },
  controls: {
    flexDirection: 'row',
    justifyContent: 'center',
    gap: 12,
  },
  button: {
    paddingHorizontal: 24,
    paddingVertical: 12,
    backgroundColor: '#E5E5EA',
    borderRadius: 8,
  },
  buttonActive: {
    backgroundColor: '#007AFF',
  },
  buttonText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#007AFF',
  },
  buffering: {
    textAlign: 'center',
    marginTop: 8,
    color: '#999',
    fontSize: 12,
  },
});
```

### 4. Integration in Your Library View

Replace your current play button with the new player:

```typescript
// app/(tabs)/library.tsx or wherever you show tracks
import { AudioPlayer } from '@/components/AudioPlayer';

export default function LibraryScreen() {
  const [selectedTrack, setSelectedTrack] = useState<Track | null>(null);
  const [tracks, setTracks] = useState<Track[]>([]);

  // ... fetch tracks ...

  return (
    <ScrollView>
      {selectedTrack && (
        <AudioPlayer
          track={selectedTrack}
          streamUrl={`${API_URL}/api/stream/${selectedTrack.id}`}
        />
      )}

      {/* Track List */}
      <View>
        {tracks.map((track) => (
          <TouchableOpacity
            key={track.id}
            onPress={() => setSelectedTrack(track)}
            style={styles.trackRow}
          >
            <View>
              <Text style={styles.trackTitle}>{track.title}</Text>
              <Text style={styles.trackArtist}>{track.artist}</Text>
            </View>
            <Text style={styles.duration}>
              {formatTime(track.duration_ms)}
            </Text>
          </TouchableOpacity>
        ))}
      </View>
    </ScrollView>
  );
}
```

### 5. Install Required Dependency

```bash
npx expo install @react-native-community/slider
```

---

## Features Breakdown

| Feature | Complexity | Priority |
|---|---|---|
| **Play/Pause Toggle** | Low | Must-have |
| **Progress Bar (Slider)** | Low | Must-have |
| **Current Time Display** | Low | Must-have |
| **Time Remaining** | Low | Must-have |
| **Seeking** | Medium | Should-have |
| **Track Info Display** | Low | Should-have |
| **Album Art Display** | Medium | Nice-to-have |
| **Skip to Next** | Medium | Nice-to-have |
| **Playback Speed Control** | Medium | Nice-to-have |
| **Repeat/Shuffle** | Medium | Nice-to-have |

**MVP includes:** Play/pause, progress bar, time display, seeking

---

## Testing

### Locally
1. Upload a track (Phase 2)
2. Select it in the library
3. Player should appear
4. Click play → audio plays
5. Progress bar updates as track plays
6. Click pause → audio stops
7. Drag progress bar → seek to new position
8. Time display updates correctly

### Edge Cases
- Track with unknown duration (set to 0, show loading)
- Very short tracks (< 1 second) — formatting should work
- Network interruption mid-stream (will error in `expo-audio`, handle gracefully)
- Switching tracks mid-playback (unload previous sound, load new one)

---

## Estimate

- **Time:** 2–3 days
  - 1 day: Component setup + play/pause + slider integration
  - 1 day: Time formatting + API updates for duration
  - 0.5–1 day: Testing + polish

---

## Known Limitations

1. **No queue/playlist** — you can only play one track at a time (add in Phase 6, Playlists & Search)
2. **No shuffle/repeat** — straightforward to add later
3. **No visualizer** — out of scope
4. **No album art** — Phase 2 issue (missing artwork_ext); can add later
5. **No track caching** — streams directly from server (fine for MVP, add later if bandwidth is a concern)

---

## Next

After Phase 3 is complete, continue to **Phase 4 (Background Playback & Lock-Screen Controls)**, then **Phase 5 (Authentication)** to secure your library.
