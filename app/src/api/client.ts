// Thin typed fetch wrapper for the Go backend. No axios/react-query per
// PROJECT.md — plain fetch is enough for this app's needs.

const API_URL = process.env.EXPO_PUBLIC_API_URL;

// The bearer token for the logged-in user. Kept as a module variable (set
// by AuthProvider via setAuthToken) so the request helpers and the
// stream/artwork URL builders don't each need it threaded through as an
// argument. null when logged out.
let authToken: string | null = null;

// Called when any authenticated request comes back 401 — i.e. the token
// expired or was revoked mid-session. AuthProvider registers a handler
// that logs the user out and bounces them to the login screen.
let onUnauthorized: (() => void) | null = null;

export function setAuthToken(token: string | null): void {
  authToken = token;
}

export function setUnauthorizedHandler(handler: (() => void) | null): void {
  onUnauthorized = handler;
}

export type Track = {
  id: number;
  original_filename: string;
  size: number;
  title: string;
  artist: string;
  album: string;
  track_number: number | null;
  duration_seconds: number | null;
  hasArtwork: boolean;
  uploaded_at: string;
};

export type Artist = {
  artist: string;
  track_count: number;
};

export type Album = {
  album: string;
  artist: string;
  track_count: number;
  representative_track_id: number;
  hasArtwork: boolean;
};

export type Playlist = {
  id: number;
  name: string;
  track_count: number;
  created_at: string;
  updated_at: string;
};

export type AuthResponse = {
  id: number;
  email: string;
  token: string;
  expiresAt: string;
};

function apiUrl(path: string): string {
  if (!API_URL) {
    throw new Error(
      'EXPO_PUBLIC_API_URL is not set. Copy app/.env.example to app/.env and set it to your server\'s LAN address.'
    );
  }
  return `${API_URL}${path}`;
}

function authHeaders(): Record<string, string> {
  return authToken ? { Authorization: `Bearer ${authToken}` } : {};
}

// withToken appends ?token= to a URL that the audio player / <Image> fetch
// directly, since those requests can't carry an Authorization header.
function withToken(url: string): string {
  if (!authToken) return url;
  const sep = url.includes('?') ? '&' : '?';
  return `${url}${sep}token=${encodeURIComponent(authToken)}`;
}

// request is fetch + the Authorization header + one place to catch an
// expired token. Only for the authenticated JSON endpoints; login/register
// call fetch directly (they have no token, and a 401 there means "bad
// credentials", not "session expired").
async function request(path: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(apiUrl(path), {
    ...init,
    headers: { ...authHeaders(), ...(init?.headers as Record<string, string> | undefined) },
  });
  if (res.status === 401) {
    onUnauthorized?.();
    throw new Error('Your session has expired. Please log in again.');
  }
  return res;
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  const res = await fetch(apiUrl('/api/auth/login'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error ?? `login failed: ${res.status}`);
  }
  return res.json();
}

export async function register(email: string, password: string): Promise<AuthResponse> {
  const res = await fetch(apiUrl('/api/auth/register'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error ?? `registration failed: ${res.status}`);
  }
  return res.json();
}

export async function getTracks(): Promise<Track[]> {
  const res = await request('/api/tracks');
  if (!res.ok) {
    throw new Error(`failed to fetch tracks: ${res.status}`);
  }
  return res.json();
}

export type TrackMetadataOverrides = {
  title?: string;
  artist?: string;
  album?: string;
  // Distinct from `artist` — shared across every track of a compilation
  // (e.g. "Various Artists") so they group into one album despite each
  // track having its own artist. Leave unset for a normal, single-artist
  // album; the server then falls back to that track's own artist.
  albumArtist?: string;
  // Position within the album — from the upload preview screen's
  // filename-parsed guess (see app/src/utils/parseFilename.ts) or a hand
  // edit. Leave unset for a well-tagged file; the server falls back to the
  // embedded tag's track number.
  trackNumber?: number;
};

// uploadTrack posts a single picked file as multipart/form-data under the
// "audio" field, matching the server's POST /api/tracks contract. Callers
// upload one file per request and loop for albums (see upload.tsx) —
// no client-side batching/parallelism per the Phase 2 spec.
//
// webFile is the DOM File from expo-document-picker's web result — on web,
// `uri` is a blob: URL, not a path, so FormData needs the real File/Blob
// (the native { uri, name, type } shape just gets stringified to
// "[object Object]" by the browser's FormData).
//
// overrides carries the upload preview screen's (possibly hand-edited)
// title/artist/album guess. A non-empty value here replaces whatever the
// server's own tag extraction/filename fallback would have produced —
// leave a field unset to let the server decide, e.g. for well-tagged files.
export async function uploadTrack(
  uri: string,
  filename: string,
  mimeType: string,
  webFile?: File,
  overrides?: TrackMetadataOverrides
): Promise<Track> {
  const form = new FormData();
  if (webFile) {
    form.append('audio', webFile, filename);
  } else {
    // React Native's FormData accepts this { uri, name, type } shape for
    // file-like values; it isn't expressible in the standard FormData type,
    // hence the cast.
    form.append('audio', { uri, name: filename, type: mimeType } as unknown as Blob);
  }
  if (overrides?.title?.trim()) form.append('title', overrides.title.trim());
  if (overrides?.artist?.trim()) form.append('artist', overrides.artist.trim());
  if (overrides?.album?.trim()) form.append('album', overrides.album.trim());
  if (overrides?.albumArtist?.trim()) form.append('album_artist', overrides.albumArtist.trim());
  if (overrides?.trackNumber) form.append('track_number', String(overrides.trackNumber));

  const res = await request('/api/tracks', {
    method: 'POST',
    body: form,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error ?? `upload failed: ${res.status}`);
  }
  return res.json();
}

export async function getArtists(): Promise<Artist[]> {
  const res = await request('/api/artists');
  if (!res.ok) {
    throw new Error(`failed to fetch artists: ${res.status}`);
  }
  return res.json();
}

export async function getArtistTracks(name: string): Promise<Track[]> {
  const res = await request(`/api/artists/${encodeURIComponent(name)}/tracks`);
  if (!res.ok) {
    throw new Error(`failed to fetch artist tracks: ${res.status}`);
  }
  return res.json();
}

export async function getAlbums(): Promise<Album[]> {
  const res = await request('/api/albums');
  if (!res.ok) {
    throw new Error(`failed to fetch albums: ${res.status}`);
  }
  return res.json();
}

// artist disambiguates same-titled albums across different artists (e.g.
// two artists each with a "Greatest Hits") — omit it to get the union.
export async function getAlbumTracks(album: string, artist?: string): Promise<Track[]> {
  const query = artist ? `?artist=${encodeURIComponent(artist)}` : '';
  const res = await request(`/api/albums/${encodeURIComponent(album)}/tracks${query}`);
  if (!res.ok) {
    throw new Error(`failed to fetch album tracks: ${res.status}`);
  }
  return res.json();
}

// --- Search -------------------------------------------------------------

// searchTracks matches q against title/artist/album (case-insensitive
// substring) on the server. A blank q comes back as [] without a round
// trip being meaningful, but callers still debounce and skip empty input.
export async function searchTracks(q: string): Promise<Track[]> {
  const res = await request(`/api/search?q=${encodeURIComponent(q)}`);
  if (!res.ok) {
    throw new Error(`search failed: ${res.status}`);
  }
  return res.json();
}

// --- Playlists ---------------------------------------------------------

export async function getPlaylists(): Promise<Playlist[]> {
  const res = await request('/api/playlists');
  if (!res.ok) {
    throw new Error(`failed to fetch playlists: ${res.status}`);
  }
  return res.json();
}

export async function createPlaylist(name: string): Promise<Playlist> {
  const res = await request('/api/playlists', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error ?? `failed to create playlist: ${res.status}`);
  }
  return res.json();
}

export async function renamePlaylist(id: number, name: string): Promise<void> {
  const res = await request(`/api/playlists/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error ?? `failed to rename playlist: ${res.status}`);
  }
}

export async function deletePlaylist(id: number): Promise<void> {
  const res = await request(`/api/playlists/${id}`, { method: 'DELETE' });
  if (!res.ok) {
    throw new Error(`failed to delete playlist: ${res.status}`);
  }
}

export async function getPlaylistTracks(id: number): Promise<Track[]> {
  const res = await request(`/api/playlists/${id}/tracks`);
  if (!res.ok) {
    throw new Error(`failed to fetch playlist tracks: ${res.status}`);
  }
  return res.json();
}

export async function addTrackToPlaylist(id: number, trackId: number): Promise<void> {
  const res = await request(`/api/playlists/${id}/tracks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ track_id: trackId }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error ?? `failed to add track: ${res.status}`);
  }
}

export async function removeTrackFromPlaylist(id: number, trackId: number): Promise<void> {
  const res = await request(`/api/playlists/${id}/tracks/${trackId}`, { method: 'DELETE' });
  if (!res.ok) {
    throw new Error(`failed to remove track: ${res.status}`);
  }
}

// reorderPlaylist sends the full track-id order; the server rejects it
// (400) unless it's exactly the playlist's current membership, so callers
// build it from the list they just rendered.
export async function reorderPlaylist(id: number, trackIds: number[]): Promise<void> {
  const res = await request(`/api/playlists/${id}/tracks`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ track_ids: trackIds }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error ?? `failed to reorder playlist: ${res.status}`);
  }
}

export function streamUrl(id: number): string {
  return withToken(apiUrl(`/api/tracks/${id}/stream`));
}

export function artworkUrl(id: number): string {
  return withToken(apiUrl(`/api/tracks/${id}/artwork`));
}
