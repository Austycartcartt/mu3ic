// Thin typed fetch wrapper for the Go backend. No axios/react-query per
// PROJECT.md — plain fetch is enough for this app's needs.

const API_URL = process.env.EXPO_PUBLIC_API_URL;

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

function apiUrl(path: string): string {
  if (!API_URL) {
    throw new Error(
      'EXPO_PUBLIC_API_URL is not set. Copy app/.env.example to app/.env and set it to your server\'s LAN address.'
    );
  }
  return `${API_URL}${path}`;
}

export async function getTracks(): Promise<Track[]> {
  const res = await fetch(apiUrl('/api/tracks'));
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

  const res = await fetch(apiUrl('/api/tracks'), {
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
  const res = await fetch(apiUrl('/api/artists'));
  if (!res.ok) {
    throw new Error(`failed to fetch artists: ${res.status}`);
  }
  return res.json();
}

export async function getArtistTracks(name: string): Promise<Track[]> {
  const res = await fetch(apiUrl(`/api/artists/${encodeURIComponent(name)}/tracks`));
  if (!res.ok) {
    throw new Error(`failed to fetch artist tracks: ${res.status}`);
  }
  return res.json();
}

export async function getAlbums(): Promise<Album[]> {
  const res = await fetch(apiUrl('/api/albums'));
  if (!res.ok) {
    throw new Error(`failed to fetch albums: ${res.status}`);
  }
  return res.json();
}

// artist disambiguates same-titled albums across different artists (e.g.
// two artists each with a "Greatest Hits") — omit it to get the union.
export async function getAlbumTracks(album: string, artist?: string): Promise<Track[]> {
  const query = artist ? `?artist=${encodeURIComponent(artist)}` : '';
  const res = await fetch(apiUrl(`/api/albums/${encodeURIComponent(album)}/tracks${query}`));
  if (!res.ok) {
    throw new Error(`failed to fetch album tracks: ${res.status}`);
  }
  return res.json();
}

export function streamUrl(id: number): string {
  return apiUrl(`/api/tracks/${id}/stream`);
}

export function artworkUrl(id: number): string {
  return apiUrl(`/api/tracks/${id}/artwork`);
}
