import { useEffect, useState } from 'react';
import { Modal, Pressable, StyleSheet, Text, TextInput, View } from 'react-native';

import { theme } from '@/theme/theme';

type Props = {
  visible: boolean;
  title: string;
  initialValue?: string;
  submitLabel?: string;
  onSubmit: (name: string) => void;
  onCancel: () => void;
};

// A centered text-prompt dialog. Used for both "New playlist" and
// "Rename playlist" — Alert.prompt is iOS-only, so this is the
// cross-platform stand-in.
export function PlaylistNameModal({
  visible,
  title,
  initialValue = '',
  submitLabel = 'Save',
  onSubmit,
  onCancel,
}: Props) {
  const [value, setValue] = useState(initialValue);

  // Reset the field whenever the dialog (re)opens.
  useEffect(() => {
    if (visible) setValue(initialValue);
  }, [visible, initialValue]);

  const trimmed = value.trim();

  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onCancel}>
      <Pressable style={styles.backdrop} onPress={onCancel}>
        <Pressable style={styles.card} onPress={() => {}}>
          <Text style={styles.title}>{title}</Text>
          <TextInput
            style={styles.input}
            value={value}
            onChangeText={setValue}
            placeholder="Playlist name"
            autoFocus
            autoCorrect={false}
            onSubmitEditing={() => trimmed && onSubmit(trimmed)}
          />
          <View style={styles.actions}>
            <Pressable style={styles.action} onPress={onCancel}>
              <Text style={styles.actionText}>Cancel</Text>
            </Pressable>
            <Pressable
              style={styles.action}
              disabled={!trimmed}
              onPress={() => onSubmit(trimmed)}
            >
              <Text style={[styles.actionText, styles.primary, !trimmed && styles.disabled]}>
                {submitLabel}
              </Text>
            </Pressable>
          </View>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.4)',
    justifyContent: 'center',
    padding: theme.spacing.xl,
  },
  card: {
    backgroundColor: theme.colors.background,
    borderRadius: theme.radii.md,
    padding: theme.spacing.lg,
    gap: theme.spacing.md,
  },
  title: {
    fontSize: theme.fontSize.lg,
    fontWeight: 'bold',
    color: theme.colors.text,
  },
  input: {
    borderWidth: 1,
    borderColor: theme.colors.border,
    borderRadius: theme.radii.sm,
    paddingVertical: theme.spacing.sm,
    paddingHorizontal: theme.spacing.md,
    fontSize: theme.fontSize.md,
  },
  actions: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    gap: theme.spacing.lg,
  },
  action: {
    paddingVertical: theme.spacing.sm,
    paddingHorizontal: theme.spacing.sm,
  },
  actionText: {
    fontSize: theme.fontSize.md,
    color: theme.colors.textMuted,
  },
  primary: {
    color: theme.colors.accent,
    fontWeight: '600',
  },
  disabled: {
    opacity: 0.4,
  },
});
