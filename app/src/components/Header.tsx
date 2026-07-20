import { Pressable, StyleSheet, Text, View } from 'react-native';

import { theme } from '@/theme/theme';

type Action = {
  label: string;
  onPress: () => void;
};

type Props = {
  title: string;
  action?: Action;
};

// No auth yet (Phase 5), so the user icon is a static placeholder — no
// avatar image, no icon library, just a letter in a circle.
export function Header({ title, action }: Props) {
  return (
    <View style={styles.container}>
      <View style={styles.avatar}>
        <Text style={styles.avatarText}>U</Text>
      </View>
      <Text style={styles.title} numberOfLines={1}>
        {title}
      </Text>
      {action && (
        <Pressable onPress={action.onPress} style={styles.action}>
          <Text style={styles.actionText}>{action.label}</Text>
        </Pressable>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing.lg,
    paddingVertical: theme.spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.border,
  },
  avatar: {
    width: 32,
    height: 32,
    borderRadius: 16,
    backgroundColor: theme.colors.accent,
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: theme.spacing.md,
  },
  avatarText: {
    color: theme.colors.onAccent,
    fontSize: theme.fontSize.md,
    fontWeight: '600',
  },
  title: {
    flex: 1,
    fontSize: theme.fontSize.xl,
    fontWeight: 'bold',
    color: theme.colors.text,
  },
  action: {
    paddingHorizontal: theme.spacing.md,
    paddingVertical: theme.spacing.sm,
  },
  actionText: {
    fontSize: theme.fontSize.md,
    fontWeight: '600',
    color: theme.colors.accent,
  },
});
