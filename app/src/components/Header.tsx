import { Alert, Pressable, StyleSheet, Text, View } from 'react-native';

import { useAuth } from '@/hooks/useAuth';
import { theme } from '@/theme/theme';

type Action = {
  label: string;
  onPress: () => void;
};

type Props = {
  title: string;
  action?: Action;
};

// The avatar is a letter in a circle (no avatar image, no icon library) —
// the logged-in user's email initial. Tapping it offers to log out.
export function Header({ title, action }: Props) {
  const { user, logout } = useAuth();
  const initial = user?.email?.[0]?.toUpperCase() ?? '?';

  function confirmLogout() {
    Alert.alert('Log out?', user?.email ?? '', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Log out', style: 'destructive', onPress: () => void logout() },
    ]);
  }

  return (
    <View style={styles.container}>
      <Pressable style={styles.avatar} onPress={confirmLogout} accessibilityLabel="Account menu">
        <Text style={styles.avatarText}>{initial}</Text>
      </Pressable>
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
