// Thin typed fetch wrapper for the Go backend. No axios/react-query per
// PROJECT.md — plain fetch is enough for this app's needs.

const API_URL = process.env.EXPO_PUBLIC_API_URL;

export type Track = {
  id: number;
  filename: string;
  size: number;
  created_at: string;
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

export function streamUrl(id: number): string {
  return apiUrl(`/api/tracks/${id}/stream`);
}
