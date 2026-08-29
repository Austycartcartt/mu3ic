import * as DocumentPicker from 'expo-document-picker';
import { Redirect, router } from 'expo-router';
import { activateKeepAwakeAsync, deactivateKeepAwake } from 'expo-keep-awake';
import { useState } from 'react';
import { FlatList, Platform, Pressable, StyleSheet, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { uploadTrack } from '@/api/client';
import { Header } from '@/components/Header';
import { PlayerDock } from '@/components/PlayerDock';
import { useAuth } from '@/hooks/useAuth';
import { theme } from '@/theme/theme';
import { parseFilename } from '@/utils/parseFilename';

type Status = 'idle' | 'preview' | 'uploading' | 'done';

// A file picked from either picker, normalized to one shape.
type PickedFile = {
  name: string;
  uri: string;
  mimeType: string;
  webFile?: File;
  relativePath?: string; // set for web folder picks, via webkitRelativePath
};

// A picked file plus its editable, filename-parsed title/artist/album/
// track-number guess — shown on the preview screen before anything is
// uploaded.
type PreviewRow = PickedFile & {
  title: string;
  artist: string;
  album: string;
  trackNumber: string; // kept as the raw text input value; parsed on upload
};

const AUDIO_EXTENSION_RE = /\.(mp3|flac|m4a|ogg|wav|aac|opus)$/i;

// Folder picking only has a real, dependency-free implementation on web
// (the browser's non-standard `webkitdirectory` input attribute). Native
// has no Expo-provided directory-tree picker, so native users stick to the
// existing flat multi-file picker below.
function pickFolderWeb(): Promise<PickedFile[]> {
  return new Promise((resolve) => {
    const input = document.createElement('input');
    input.type = 'file';
    (input as unknown as { webkitdirectory: boolean }).webkitdirectory = true;
    input.multiple = true;
    input.addEventListener('change', () => {
      const files = Array.from(input.files ?? []).filter(
        (f) => f.type.startsWith('audio/') || AUDIO_EXTENSION_RE.test(f.name)
      );
      resolve(
        files.map((f) => ({
          name: f.name,
          uri: '',
          mimeType: f.type || 'application/octet-stream',
          webFile: f,
          relativePath: (f as unknown as { webkitRelativePath?: string }).webkitRelativePath || undefined,
        }))
      );
    });
    input.click();
  });
}

// Runs the filename parser over each picked file to build the preview
// screen's starting point — a field is only pre-filled when the parser is
// confident, so an unrecognized filename never silently overrides a real
// embedded tag (the user can still type into a blank field to force one).
function buildPreviewRows(files: PickedFile[]): PreviewRow[] {
  return files.map((file) => {
    const guess = parseFilename(file.name);
    return {
      ...file,
      title: guess.confident ? (guess.title ?? '') : '',
      artist: guess.confident ? (guess.artist ?? '') : '',
      album: guess.confident ? (guess.album ?? '') : '',
      trackNumber: guess.confident && guess.trackNumber !== undefined ? String(guess.trackNumber) : '',
    };
  });
}

export default function UploadScreen() {
  const { token, isLoading } = useAuth();
  const [status, setStatus] = useState<Status>('idle');
  const [previewRows, setPreviewRows] = useState<PreviewRow[]>([]);
  // Applies to every file in this batch, not per-row like title/artist/
  // album below — a various-artists compilation needs one shared album
  // artist (e.g. "Various Artists") so its differently-artist-tagged
  // tracks group into a single album instead of one per track artist.
  // Left blank, the server falls back to each track's own artist.
  const [albumArtist, setAlbumArtist] = useState('');
  const [current, setCurrent] = useState(0);
  const [total, setTotal] = useState(0);
  const [currentName, setCurrentName] = useState('');
  const [failures, setFailures] = useState<string[]>([]);

  async function pickFiles() {
    const result = await DocumentPicker.getDocumentAsync({
      type: 'audio/*',
      multiple: true,
      copyToCacheDirectory: true,
    });
    if (result.canceled) {
      return;
    }
    const files: PickedFile[] = result.assets.map((asset) => ({
      name: asset.name,
      uri: asset.uri,
      mimeType: asset.mimeType ?? 'application/octet-stream',
      webFile: asset.file,
    }));
    setAlbumArtist('');
    setPreviewRows(buildPreviewRows(files));
    setStatus('preview');
  }

  async function pickFolder() {
    const files = await pickFolderWeb();
    if (files.length === 0) {
      return;
    }
    setAlbumArtist('');
    setPreviewRows(buildPreviewRows(files));
    setStatus('preview');
  }

  function updateRow(index: number, field: 'title' | 'artist' | 'album' | 'trackNumber', value: string) {
    setPreviewRows((rows) => rows.map((row, i) => (i === index ? { ...row, [field]: value } : row)));
  }

  async function confirmUpload() {
    const rows = previewRows;
    setStatus('uploading');
    setTotal(rows.length);
    setCurrent(0);
    setFailures([]);

    await activateKeepAwakeAsync();
    try {
      const failed: string[] = [];
      // Uploaded sequentially, not in parallel, per the Phase 2 spec — one
      // fetch + FormData at a time keeps memory/network use predictable
      // for album-scale (10-15 track) uploads.
      for (let i = 0; i < rows.length; i++) {
        const row = rows[i];
        setCurrent(i + 1);
        setCurrentName(row.name);
        try {
          const trackNumber = parseInt(row.trackNumber, 10);
          await uploadTrack(row.uri, row.name, row.mimeType, row.webFile, {
            title: row.title,
            artist: row.artist,
            album: row.album,
            albumArtist,
            trackNumber: Number.isNaN(trackNumber) ? undefined : trackNumber,
          });
        } catch {
          failed.push(row.name);
        }
      }
      setFailures(failed);
      setStatus('done');
    } finally {
      await deactivateKeepAwake();
    }
  }

  // This screen sits outside (tabs), so it needs its own auth guard.
  if (!isLoading && !token) return <Redirect href="/login" />;

  return (
    <SafeAreaView style={styles.container}>
      <Header title="Upload" action={{ label: 'Back', onPress: () => router.back() }} />

      {status === 'idle' && (
        <View style={styles.content}>
          <Pressable style={styles.button} onPress={pickFiles}>
            <Text style={styles.buttonText}>Choose audio files</Text>
          </Pressable>
          {Platform.OS === 'web' && (
            <Pressable style={[styles.button, styles.secondaryButton]} onPress={pickFolder}>
              <Text style={styles.buttonText}>Choose a folder</Text>
            </Pressable>
          )}
        </View>
      )}

      {status === 'preview' && (
        <>
          <View style={styles.albumArtistRow}>
            <Text style={styles.albumArtistLabel}>
              Album Artist (optional — for a various-artists compilation, e.g. &quot;Various Artists&quot;)
            </Text>
            <TextInput
              style={styles.input}
              placeholder="Leave blank for a normal, single-artist album"
              value={albumArtist}
              onChangeText={setAlbumArtist}
            />
          </View>
          <FlatList
            data={previewRows}
            keyExtractor={(_, index) => index.toString()}
            contentContainerStyle={styles.previewList}
            renderItem={({ item, index }) => (
              <View style={styles.previewRow}>
                <Text style={styles.previewFilename} numberOfLines={1}>
                  {item.relativePath ?? item.name}
                </Text>
                <TextInput
                  style={styles.input}
                  placeholder="Title"
                  value={item.title}
                  onChangeText={(v) => updateRow(index, 'title', v)}
                />
                <TextInput
                  style={styles.input}
                  placeholder="Artist"
                  value={item.artist}
                  onChangeText={(v) => updateRow(index, 'artist', v)}
                />
                <TextInput
                  style={styles.input}
                  placeholder="Album"
                  value={item.album}
                  onChangeText={(v) => updateRow(index, 'album', v)}
                />
                <TextInput
                  style={styles.input}
                  placeholder="Track #"
                  value={item.trackNumber}
                  onChangeText={(v) => updateRow(index, 'trackNumber', v)}
                  keyboardType="number-pad"
                />
              </View>
            )}
          />
          <View style={styles.previewActions}>
            <Pressable style={[styles.button, styles.secondaryButton]} onPress={() => setStatus('idle')}>
              <Text style={styles.buttonText}>Cancel</Text>
            </Pressable>
            <Pressable style={styles.button} onPress={confirmUpload}>
              <Text style={styles.buttonText}>Upload {previewRows.length} files</Text>
            </Pressable>
          </View>
        </>
      )}

      {status === 'uploading' && (
        <View style={styles.content}>
          <Text style={styles.status}>
            Uploading {current} of {total} — {currentName}
          </Text>
        </View>
      )}

      {status === 'done' && (
        <View style={styles.content}>
          <Text style={styles.status}>
            {total - failures.length} uploaded, {failures.length} failed
          </Text>
          {failures.map((name) => (
            <Text key={name} style={styles.failure}>
              {name}
            </Text>
          ))}
          <Pressable style={styles.button} onPress={() => setStatus('idle')}>
            <Text style={styles.buttonText}>Upload more</Text>
          </Pressable>
        </View>
      )}

      <PlayerDock />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  content: {
    flex: 1,
    padding: theme.spacing.lg,
    justifyContent: 'center',
    alignItems: 'center',
  },
  button: {
    backgroundColor: theme.colors.accent,
    paddingVertical: theme.spacing.md,
    paddingHorizontal: theme.spacing.xl,
    borderRadius: theme.radii.md,
    marginTop: theme.spacing.lg,
  },
  secondaryButton: {
    backgroundColor: theme.colors.textMuted,
  },
  buttonText: {
    color: theme.colors.onAccent,
    fontSize: theme.fontSize.lg,
  },
  status: {
    fontSize: theme.fontSize.lg,
    textAlign: 'center',
  },
  failure: {
    fontSize: theme.fontSize.sm,
    color: theme.colors.danger,
    textAlign: 'center',
    marginTop: theme.spacing.xs,
  },
  albumArtistRow: {
    paddingHorizontal: theme.spacing.lg,
    paddingTop: theme.spacing.lg,
  },
  albumArtistLabel: {
    fontSize: theme.fontSize.sm,
    color: theme.colors.textMuted,
    marginBottom: theme.spacing.xs,
  },
  previewList: {
    padding: theme.spacing.lg,
  },
  previewRow: {
    borderWidth: 1,
    borderColor: theme.colors.border,
    borderRadius: theme.radii.md,
    padding: theme.spacing.md,
    marginBottom: theme.spacing.md,
  },
  previewFilename: {
    fontSize: theme.fontSize.sm,
    color: theme.colors.textMuted,
    marginBottom: theme.spacing.sm,
  },
  input: {
    borderWidth: 1,
    borderColor: theme.colors.border,
    borderRadius: theme.radii.sm,
    paddingVertical: theme.spacing.xs,
    paddingHorizontal: theme.spacing.sm,
    marginBottom: theme.spacing.xs,
    fontSize: theme.fontSize.md,
  },
  previewActions: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    padding: theme.spacing.lg,
  },
});
