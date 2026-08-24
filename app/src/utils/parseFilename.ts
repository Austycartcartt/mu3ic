// Best-effort filename -> {artist, album, title} guesser for messy library
// filenames (e.g. "Artist - Album - 01 Title.mp3" or
// "Album-01-001-Artist-Title.wav"). This is heuristic, not a general
// filename grammar — the upload preview screen shows its guesses in
// editable fields, and only pre-fills them when `confident` is true, so an
// unmatched filename never silently overrides real embedded tags.

export type ParsedGuess = {
  artist?: string;
  album?: string;
  title?: string;
  trackNumber?: number;
  confident: boolean;
};

function isNumeric(segment: string): boolean {
  return /^\d+$/.test(segment);
}

function stripExtension(filename: string): string {
  return filename.replace(/\.[^./\\]+$/, '');
}

export function parseFilename(filename: string): ParsedGuess {
  const segments = stripExtension(filename)
    .split('-')
    .map((s) => s.trim())
    .filter((s) => s.length > 0);

  // "Artist - Album - 01 Title" (padded " - " and bare "-" both normalize
  // to this once each segment is trimmed).
  if (segments.length === 3) {
    const [artist, album, trailing] = segments;
    const match = trailing.match(/^(\d+)\s+(.+)$/);
    const trackNumber = match ? parseInt(match[1], 10) : undefined;
    const title = match ? match[2].trim() : trailing;
    return { artist, album, title, trackNumber, confident: true };
  }

  // "Album-01-001-Artist-Title" — two numeric index/track segments between
  // the album and artist.
  if (segments.length === 5 && isNumeric(segments[1]) && isNumeric(segments[2])) {
    const [album, , trackNumStr, artist, title] = segments;
    return { album, artist, title, trackNumber: parseInt(trackNumStr, 10), confident: true };
  }

  // Generic fallback: drop numeric segments as a candidate track number,
  // then use whatever's left if it cleanly maps to [artist, title] or
  // [artist, album, title].
  const numeric = segments.filter(isNumeric);
  const nonNumeric = segments.filter((s) => !isNumeric(s));
  const trackNumber = numeric.length > 0 ? parseInt(numeric[0], 10) : undefined;

  if (nonNumeric.length === 2) {
    const [artist, title] = nonNumeric;
    return { artist, title, trackNumber, confident: true };
  }
  if (nonNumeric.length === 3) {
    const [artist, album, title] = nonNumeric;
    return { artist, album, title, trackNumber, confident: true };
  }

  return { confident: false };
}
