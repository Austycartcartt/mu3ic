# Phase 4: Background Playback & Lock-Screen Controls

**Status:** Planned (start date logged 2026-07-17 in Notion, not yet started in-repo)

## Goal

Enable audio playback to persist when the Expo app backgrounds, and provide native lock-screen controls (play/pause, next/previous, track metadata display).

---

## Why This Matters

The vertical slice (Phase 1) proved connectivity. Phase 2 added upload + metadata. But right now, as soon as the app backgrounds, `expo-audio` stops playback — which breaks the core use case of a personal streaming app.

Lock-screen controls and status notifications are expected on both iOS and Android for any serious audio app. Users expect to control playback without re-opening the app.

---

## The Constraint: Custom Development Build

**This phase requires a custom Expo development build.** You cannot use Expo Go.

### Why?

- `expo-audio` (used for playback) does not support background playback in Expo Go
- Background audio requires native platform permissions and integration (iOS: Audio Session configuration + Info.plist; Android: foreground service)
- Lock-screen controls require native bridge implementation

### Implication

You will need to generate a development build via **EAS Build** (Expo's cloud build service) or build locally with Xcode/Android Studio.

**This is not a blocker for proceeding — it's just a known gate.**

---

## Apple Developer Account Timing

**Do you need a paid Apple Developer Program membership to proceed?**

- **For local iOS development / simulator:** No. Build locally with Xcode; run on simulator.
- **For physical iOS device testing:** Yes, a paid account ($99/year) is required to generate provisioning profiles and code-sign for device deployment.

**Recommendation:** Start with iOS simulator testing (free) to validate the architecture. Migrate to physical device when needed.

---

## Technical Breakdown

### 1. iOS Background Playback

**Audio Session Setup:**

```swift
// In ios/Podfile or native module
import AVFoundation

let audioSession = AVAudioSession.sharedInstance()
try audioSession.setCategory(
    .playback,
    mode: .default,
    options: [.duckOthers]
)
try audioSession.setActive(true)
```

**Info.plist Entries:**

```xml
<key>UIBackgroundModes</key>
<array>
    <key>audio</key>
</array>
```

**Flow:**
1. User initiates playback in Expo app
2. `expo-audio` holds the audio stream open
3. App backgrounds → iOS suspends JavaScript but keeps native audio layer alive
4. Lock-screen UI (native, not React) shows playback controls
5. User taps controls on lock screen → native event triggers JavaScript callback → update playback state

### 2. Android Background Playback

**Foreground Service:**

Android requires a visible notification (foreground service) to keep playback alive in the background.

```kotlin
// ExoPlayer or similar
startForegroundService(Intent(this, AudioPlaybackService::class.java))
```

**Permission in AndroidManifest.xml:**

```xml
<uses-permission android:name="android.permission.FOREGROUND_SERVICE_MEDIA_PLAYBACK" />
```

**Flow:** Similar to iOS — native service holds the audio session; native controls drive playback state.

### 3. Lock-Screen / System Controls

Both platforms provide native APIs for media controls:

- **iOS:** `MPRemoteCommandCenter` — register play/pause/next/previous commands
- **Android:** `MediaSession` — publish playback state, register control callbacks

These are platform-specific and require native modules or Expo's `expo-media-player` (if available) or community packages like `react-native-track-player`.

**Note:** `expo-audio` may not have full native bridge support for lock-screen controls. If it doesn't, you'll need either:
1. A **custom native module** (Expo Config Plugin)
2. A community package that wraps these APIs (e.g., `react-native-track-player`, which is NOT part of Expo)

### 4. Metadata Display

Lock-screen and notification metadata (track title, artist, album art):

- Fetch from the tracks table (already in Postgres)
- Pass to native layer via bridge
- Native layer renders in lock-screen UI

---

## Implementation Roadmap

### 4a: Foundation (Custom Dev Build)

1. **Set up EAS CLI locally**
   - Install `eas-cli` globally
   - Authenticate with an Expo account
   - Link project: `eas build:configure`
2. **Generate a development build**
   - `eas build --platform ios --profile preview` (or `--profile development` for faster iteration)
   - Deploy to simulator or (if you have an Apple Developer account) to device
3. **Test that background audio works**
   - Start playback in the app
   - Background the app
   - Confirm audio continues
   - Foreground the app → playback is still active

### 4b: Lock-Screen Controls

1. **Identify the native module / package**
   - Option A: Use `react-native-track-player` (community, battle-tested, but adds dependency)
   - Option B: Build a custom native module via Expo Config Plugin (learning experience, more control)
   - **Recommend Option A for now** — less cognitive load, proven to work with Expo
2. **Integrate into current playback flow**
   - Wrap current `expo-audio` calls or replace with `react-native-track-player`
   - Update player component to register remote controls
   - Test on simulator/device
3. **Connect to backend**
   - Lock-screen controls should send track updates via the same API calls (no auth needed for MVP)
   - Metadata is already in Postgres — pass it to native layer when playback starts

### 4c: Notifications & Polish

1. **Foreground notification** (Android) showing current track
2. **Status notifications** (both platforms) showing playback progress
3. **Artwork caching** — avoid fetching album art on every control tap

---

## Open Questions

1. **React Native Track Player vs. custom native module?**
   - Track player is faster to ship; custom module is a learning opportunity.
   - Recommend: **start with track player, revisit custom module post-MVP if needed.**
2. **What about streaming interruptions?** (calls, other audio)
   - `expo-audio` should handle this automatically (iOS Audio Session duck/resume)
   - May need explicit pause logic in the Expo app
3. **How to cache album artwork?**
   - If `artwork_ext` is missing (current issue), construct a fallback (solid color, initials)
   - For cached images, use `expo-file-system` + local filesystem
4. **Can we deploy the dev build to a friend's device without an Apple Developer account?**
   - No. Physical device requires a provisioning profile, which requires a paid account.
   - Workaround: they can use a local iOS simulator if they have a Mac, or Android emulator (free).

---

## Known Blockers

1. **Missing metadata from Phase 2** (artist, album, duration, artwork_ext)
   - Fix test data before this phase starts, or ensure fallback logic is in place for missing fields
2. **No custom native module experience yet**
   - If choosing the custom module route, will need to learn Xcode project structure + Kotlin/Swift basics
   - Recommend: track player is the safer first step

---

## Success Criteria

- Audio playback persists when app is backgrounded
- Lock-screen controls (play/pause, skip) are visible and responsive
- Track metadata (title, artist) displays on lock screen
- Playback state is synchronized between app and lock-screen UI
- Background playback survives app restart

---

## Timeline Estimate

- **4a (Dev Build + Background Audio):** 2–3 days
- **4b (Lock-Screen Controls):** 3–5 days
- **4c (Notifications & Polish):** 2–3 days (optional for MVP)

**Total estimate: 1–2 weeks** depending on setup complexity and platform-specific bugs.

---

## Next Steps

1. Decide: Apple Developer account now, or start with simulator?
2. Set up EAS CLI + first dev build; document the process, capture any errors
3. Test background audio with a minimal start/stop/background test before adding lock-screen controls
4. Return to Phase 2 metadata gaps (artist, album, duration, artwork) if time permits
