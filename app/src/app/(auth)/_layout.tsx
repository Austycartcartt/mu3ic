import { Redirect, Stack } from 'expo-router';

import { useAuth } from '@/hooks/useAuth';

// The login / register stack. Already-authenticated users never see it —
// they're redirected into the app.
export default function AuthLayout() {
  const { token, isLoading } = useAuth();

  if (isLoading) return null;
  if (token) return <Redirect href="/" />;

  return <Stack screenOptions={{ headerShown: false }} />;
}
