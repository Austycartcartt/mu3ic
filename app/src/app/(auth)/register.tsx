import { Link, router } from 'expo-router';
import { useState } from 'react';
import { Pressable, StyleSheet, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { useAuth } from '@/hooks/useAuth';
import { theme } from '@/theme/theme';

// Keep in sync with the server's minPasswordLen (internal/api/auth.go).
const MIN_PASSWORD_LEN = 8;

export default function RegisterScreen() {
  const { register } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit() {
    if (password.length < MIN_PASSWORD_LEN) {
      setError(`Password must be at least ${MIN_PASSWORD_LEN} characters.`);
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await register(email.trim(), password);
      router.replace('/');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.content}>
        <Text style={styles.title}>Create an account</Text>

        {error && (
          <View style={styles.errorBanner}>
            <Text style={styles.errorText}>{error}</Text>
          </View>
        )}

        <TextInput
          style={styles.input}
          placeholder="Email"
          value={email}
          onChangeText={setEmail}
          keyboardType="email-address"
          autoCapitalize="none"
          autoComplete="email"
          autoCorrect={false}
        />
        <TextInput
          style={styles.input}
          placeholder={`Password (min ${MIN_PASSWORD_LEN} characters)`}
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoCapitalize="none"
        />

        <Pressable style={[styles.button, busy && styles.buttonDisabled]} onPress={submit} disabled={busy}>
          <Text style={styles.buttonText}>{busy ? 'Creating…' : 'Register'}</Text>
        </Pressable>

        <Link href="/login" style={styles.link}>
          Already have an account? Log in
        </Link>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: theme.colors.background,
  },
  content: {
    flex: 1,
    padding: theme.spacing.xl,
    justifyContent: 'center',
    gap: theme.spacing.sm,
  },
  title: {
    fontSize: theme.fontSize.xl,
    fontWeight: 'bold',
    color: theme.colors.text,
    marginBottom: theme.spacing.md,
  },
  input: {
    borderWidth: 1,
    borderColor: theme.colors.border,
    borderRadius: theme.radii.sm,
    paddingVertical: theme.spacing.sm,
    paddingHorizontal: theme.spacing.md,
    fontSize: theme.fontSize.md,
  },
  button: {
    backgroundColor: theme.colors.accent,
    paddingVertical: theme.spacing.md,
    borderRadius: theme.radii.md,
    marginTop: theme.spacing.md,
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  buttonText: {
    color: theme.colors.onAccent,
    fontSize: theme.fontSize.lg,
    textAlign: 'center',
  },
  link: {
    color: theme.colors.accent,
    fontSize: theme.fontSize.md,
    textAlign: 'center',
    marginTop: theme.spacing.lg,
  },
  errorBanner: {
    padding: theme.spacing.md,
    backgroundColor: theme.colors.dangerBackground,
    borderRadius: theme.radii.sm,
  },
  errorText: {
    color: theme.colors.danger,
  },
});
