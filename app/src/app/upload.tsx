import * as DocumentPicker from 'expo-document-picker';
import { activateKeepAwakeAsync, deactivateKeepAwake } from 'expo-keep-awake';
import { useState } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { uploadTrack } from '@/api/client';
import { Header } from '@/components/Header';
import { theme } from '@/theme/theme';

type Status = 'idle' | 'uploading' | 'done';

export default function UploadScreen() {
  const [status, setStatus] = useState<Status>('idle');
  const [current, setCurrent] = useState(0);
  const [total, setTotal] = useState(0);
  const [currentName, setCurrentName] = useState('');
  const [failures, setFailures] = useState<string[]>([]);

  async function pickAndUpload() {
    const result = await DocumentPicker.getDocumentAsync({
      type: 'audio/*',
      multiple: true,
      copyToCacheDirectory: true,
    });
    if (result.canceled) {
      return;
    }

    const assets = result.assets;
    setStatus('uploading');
    setTotal(assets.length);
    setCurrent(0);
    setFailures([]);

    await activateKeepAwakeAsync();
    try {
      const failed: string[] = [];
      // Uploaded sequentially, not in parallel, per the Phase 2 spec — one
      // fetch + FormData at a time keeps memory/network use predictable
      // for album-scale (10-15 track) uploads.
      for (let i = 0; i < assets.length; i++) {
        const asset = assets[i];
        setCurrent(i + 1);
        setCurrentName(asset.name);
        try {
          await uploadTrack(asset.uri, asset.name, asset.mimeType ?? 'application/octet-stream');
        } catch {
          failed.push(asset.name);
        }
      }
      setFailures(failed);
      setStatus('done');
    } finally {
      await deactivateKeepAwake();
    }
  }

  return (
    <SafeAreaView style={styles.container}>
      <Header title="Upload" />
      <View style={styles.content}>
        {status === 'idle' && (
          <Pressable style={styles.button} onPress={pickAndUpload}>
            <Text style={styles.buttonText}>Choose audio files</Text>
          </Pressable>
        )}

        {status === 'uploading' && (
          <Text style={styles.status}>
            Uploading {current} of {total} — {currentName}
          </Text>
        )}

        {status === 'done' && (
          <View>
            <Text style={styles.status}>
              {total - failures.length} uploaded, {failures.length} failed
            </Text>
            {failures.map((name) => (
              <Text key={name} style={styles.failure}>
                {name}
              </Text>
            ))}
            <Pressable style={styles.button} onPress={pickAndUpload}>
              <Text style={styles.buttonText}>Upload more</Text>
            </Pressable>
          </View>
        )}
      </View>
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
});
