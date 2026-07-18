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
  duration_seconds: number | null;
  hasArtwork: boolean;
  uploaded_at: string;
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

// uploadTrack posts a single picked file as multipart/form-data under the
// "audio" field, matching the server's POST /api/tracks contract. Callers
// upload one file per request and loop for albums (see upload.tsx) —
// no client-side batching/parallelism per the Phase 2 spec.
export async function uploadTrack(uri: string, filename: string, mimeType: string): Promise<Track> {
  const form = new FormData();
  // React Native's FormData accepts this { uri, name, type } shape for
  // file-like values; it isn't expressible in the standard FormData type,
  // hence the cast.
  form.append('audio', { uri, name: filename, type: mimeType } as unknown as Blob);

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

export function streamUrl(id: number): string {
  return apiUrl(`/api/tracks/${id}/stream`);
}

export function artworkUrl(id: number): string {
  return apiUrl(`/api/tracks/${id}/artwork`);
}
