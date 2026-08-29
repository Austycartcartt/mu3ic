# Phase 5: Authentication

**Status:** Complete (2026-08-27)

**What shipped vs. this plan:** login is by **email**, not username (`users.email UNIQUE`); `users.id` is `BIGSERIAL`, not UUID (matches `tracks.id`); the stream/artwork routes are `/api/tracks/{id}/{stream,artwork}`, not `/api/stream/:id`. One `withAuth` middleware accepts the token from either the `Authorization: Bearer` header or a `?token=` query param. Registration is open. See the "JWT auth" entry in [DECISIONS.md](DECISIONS.md) for the full record.

## Goal

Implement user accounts and session management. Multiple users should be able to log in, have their own libraries, and only stream their own music.

---

## The Core Challenge

**Simple login/logout?** Easy.

**Attaching auth to stream URLs?** That's the trick.

### Why It's Tricky

Right now, the `/api/stream/:id` endpoint uses HTTP range requests, which works like this:

```
Expo app (audio player)
    ↓
Makes HTTP GET request to /api/stream/track-uuid
    ↓
Go server
    ↓
Returns audio bytes via http.ServeContent
```

**The problem:** Once the audio player starts fetching, we can't inject custom HTTP headers. The player controls the request headers, not our app code.

**So how do we authenticate?** We need the token *in the URL itself*, as a query parameter:

```
GET /api/stream/track-uuid?token=eyJhbGc...
```

This way, the server can:
1. Extract the token from the URL
2. Verify it (check signature, check it hasn't expired)
3. Check that the user owns the track
4. Serve the audio (or 403 if unauthorized)

---

## Architecture Decision: Stateless Tokens (JWT)

**Why JWT?**
- **Stateless:** No need to store sessions in the database (simpler)
- **Self-contained:** Token includes user info + expiration, server just verifies the signature
- **URL-friendly:** Can be embedded in query params without issues
- **Scalable:** If you ever add multiple servers, JWTs work without shared session storage

**Why not session cookies?**
- Sessions require DB lookups on every request
- Harder to embed in stream URLs
- Less elegant for this use case

**Why not OAuth?**
- Overkill for a personal app where you're both the server and the only user (initially)
- Add OAuth later if you want to share libraries with friends

*(This decision should be promoted into [DECISIONS.md](DECISIONS.md) once implementation confirms it holds.)*

---

## Implementation Plan

### Backend (Go)

#### 1. Database Schema

Add a `users` table:

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
```

Update `tracks` table to add a user reference:

```sql
ALTER TABLE tracks ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;
```

#### 2. User Management Endpoints

**POST /api/auth/register**

```json
Request:
{
  "username": "austy",
  "password": "secure-password"
}

Response (201):
{
  "id": "uuid",
  "username": "austy",
  "token": "eyJhbGc..."
}
```

**POST /api/auth/login**

```json
Request:
{
  "username": "austy",
  "password": "secure-password"
}

Response (200):
{
  "id": "uuid",
  "username": "austy",
  "token": "eyJhbGc...",
  "expiresAt": "2026-07-18T12:00:00Z"
}
```

#### 3. JWT Implementation

Use `golang-jwt/jwt/v5` (lightweight, stdlib-compatible):

```go
import "github.com/golang-jwt/jwt/v5"

type Claims struct {
  UserID   string `json:"user_id"`
  Username string `json:"username"`
  jwt.RegisteredClaims
}

// Generate token
func GenerateToken(userID, username string, secret string) (string, error) {
  claims := Claims{
    UserID:   userID,
    Username: username,
    RegisteredClaims: jwt.RegisteredClaims{
      ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24-hour expiry
      IssuedAt:  jwt.NewNumericDate(time.Now()),
    },
  }

  token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
  return token.SignedString([]byte(secret))
}

// Verify token
func VerifyToken(tokenString string, secret string) (*Claims, error) {
  claims := &Claims{}
  token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
    return []byte(secret), nil
  })

  if err != nil || !token.Valid {
    return nil, err
  }

  return claims, nil
}
```

#### 4. Protect Endpoints

**GET /api/tracks** (list user's tracks)

```go
// Extract token from query or header
token := r.URL.Query().Get("token")
if token == "" {
  token = r.Header.Get("Authorization") // fallback to Authorization header
}

claims, err := VerifyToken(token, SECRET)
if err != nil {
  http.Error(w, "Unauthorized", http.StatusUnauthorized)
  return
}

// Query only this user's tracks
rows, err := db.Query(
  "SELECT id, title, artist, album, duration FROM tracks WHERE user_id = $1",
  claims.UserID,
)
```

**GET /api/stream/:id** (stream user's track)

```go
token := r.URL.Query().Get("token")
claims, err := VerifyToken(token, SECRET)
if err != nil {
  http.Error(w, "Unauthorized", http.StatusUnauthorized)
  return
}

// Check user owns this track
var trackUserID string
err = db.QueryRow(
  "SELECT user_id FROM tracks WHERE id = $1",
  trackID,
).Scan(&trackUserID)

if trackUserID != claims.UserID {
  http.Error(w, "Forbidden", http.StatusForbidden)
  return
}

// Serve the file
http.ServeContent(w, r, filename, modTime, file)
```

**POST /api/tracks** (upload stays the same, but now associates track with logged-in user)

```go
// Extract user from request (via token or header)
claims, err := VerifyToken(token, SECRET)
// Insert track with user_id = claims.UserID
```

#### 5. Config & Secrets

Store the JWT secret securely:
- **Development:** Environment variable `JWT_SECRET`
- **Production:** Use a secrets manager (e.g., HashiCorp Vault, AWS Secrets Manager)

```go
secret := os.Getenv("JWT_SECRET")
if secret == "" {
  secret = "dev-secret-change-in-production"
}
```

---

### Frontend (Expo)

#### 1. Secure Token Storage

Use `expo-secure-store` to persist the token:

```typescript
import * as SecureStore from 'expo-secure-store';

// After login
await SecureStore.setItemAsync('auth_token', token);

// On app launch
const token = await SecureStore.getItemAsync('auth_token');
if (token) {
  // Auto-login (verify token is still valid)
}

// On logout
await SecureStore.deleteItemAsync('auth_token');
```

#### 2. Auth Context

Create a custom hook for auth state:

```typescript
import { createContext, useContext, useState, useEffect } from 'react';
import * as SecureStore from 'expo-secure-store';

type AuthContextType = {
  user: { id: string; username: string } | null;
  token: string | null;
  login: (username: string, password: string) => Promise<void>;
  register: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  isLoading: boolean;
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<{ id: string; username: string } | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Check for existing token on app launch
  useEffect(() => {
    async function bootstrap() {
      try {
        const savedToken = await SecureStore.getItemAsync('auth_token');
        if (savedToken) {
          setToken(savedToken);
          // Optionally verify token is valid by fetching user profile
        }
      } catch (e) {
        console.error('Failed to restore token', e);
      } finally {
        setIsLoading(false);
      }
    }

    bootstrap();
  }, []);

  const login = async (username: string, password: string) => {
    const res = await fetch(`${API_URL}/api/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    });

    if (!res.ok) throw new Error('Login failed');

    const data = await res.json();
    setToken(data.token);
    setUser({ id: data.id, username: data.username });
    await SecureStore.setItemAsync('auth_token', data.token);
  };

  const register = async (username: string, password: string) => {
    const res = await fetch(`${API_URL}/api/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    });

    if (!res.ok) throw new Error('Registration failed');

    const data = await res.json();
    setToken(data.token);
    setUser({ id: data.id, username: data.username });
    await SecureStore.setItemAsync('auth_token', data.token);
  };

  const logout = async () => {
    setToken(null);
    setUser(null);
    await SecureStore.deleteItemAsync('auth_token');
  };

  return (
    <AuthContext.Provider value={{ user, token, login, register, logout, isLoading }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be called within AuthProvider');
  return context;
}
```

#### 3. Login/Register Screens

Build simple forms using Expo Router + React Native:

```typescript
// app/(auth)/login.tsx
import { useState } from 'react';
import { View, TextInput, TouchableOpacity, Text } from 'react-native';
import { useAuth } from '@/context/AuthContext';
import { useRouter } from 'expo-router';

export default function LoginScreen() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const { login } = useAuth();
  const router = useRouter();

  const handleLogin = async () => {
    try {
      await login(username, password);
      router.replace('/(tabs)/library');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    }
  };

  return (
    <View style={{ flex: 1, padding: 20, justifyContent: 'center' }}>
      <TextInput
        placeholder="Username"
        value={username}
        onChangeText={setUsername}
        style={{ borderWidth: 1, padding: 10, marginBottom: 10 }}
      />
      <TextInput
        placeholder="Password"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        style={{ borderWidth: 1, padding: 10, marginBottom: 10 }}
      />
      {error && <Text style={{ color: 'red', marginBottom: 10 }}>{error}</Text>}
      <TouchableOpacity
        onPress={handleLogin}
        style={{ backgroundColor: '#007AFF', padding: 10, borderRadius: 5 }}
      >
        <Text style={{ color: 'white', textAlign: 'center' }}>Login</Text>
      </TouchableOpacity>
    </View>
  );
}
```

#### 4. Attach Token to Requests

When fetching tracks, attach the token as a query parameter:

```typescript
const { token } = useAuth();

const fetchTracks = async () => {
  const res = await fetch(`${API_URL}/api/tracks?token=${token}`);
  const tracks = await res.json();
  setTracks(tracks);
};
```

For audio streaming, construct the URL with the token:

```typescript
const streamUrl = `${API_URL}/api/stream/${trackId}?token=${token}`;
// Pass streamUrl to audio player
```

#### 5. Navigation Structure

Conditionally show login screen or main app based on auth state:

```typescript
// app/_layout.tsx (Expo Router root layout)
import { AuthProvider, useAuth } from '@/context/AuthContext';
import { Stack } from 'expo-router';

function RootLayoutNav() {
  const { token, isLoading } = useAuth();

  if (isLoading) {
    return <LoadingScreen />;
  }

  return (
    <Stack>
      {!token ? (
        <>
          <Stack.Screen
            name="(auth)/login"
            options={{ headerShown: false }}
          />
          <Stack.Screen
            name="(auth)/register"
            options={{ headerShown: false }}
          />
        </>
      ) : (
        <Stack.Screen
          name="(tabs)"
          options={{ headerShown: false }}
        />
      )}
    </Stack>
  );
}

export default function RootLayout() {
  return (
    <AuthProvider>
      <RootLayoutNav />
    </AuthProvider>
  );
}
```

---

## Security Considerations

### 1. Password Hashing

Never store plaintext passwords. Use `bcrypt`:

```go
import "golang.org/x/crypto/bcrypt"

// Hash password at registration
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

// Verify password at login
err := bcrypt.CompareHashAndPassword(passwordHash, []byte(password))
```

### 2. Token Expiration

JWTs should have a short lifespan (24 hours for MVP). After expiration, force re-login.

### 3. HTTPS (Production)

In production, always use HTTPS to prevent token interception. On a home server, use Let's Encrypt.

### 4. Token in URL vs. Header

**Trade-off:**
- **URL:** Works with range requests, but tokens appear in browser history/logs
- **Header:** Cleaner, but can't be used for audio streaming

**Solution:** For MVP, tokens in URL are fine. If security is critical later, use a session-based approach (server issues short-lived tokens for streaming only).

---

## Testing Strategy

### Backend

```bash
# Register
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"austy","password":"password123"}'

# Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"austy","password":"password123"}'

# List tracks (with token)
curl http://localhost:8080/api/tracks?token=<YOUR_TOKEN>

# Stream track (with token)
curl http://localhost:8080/api/stream/track-uuid?token=<YOUR_TOKEN> \
  -H "Range: bytes=0-1023"
```

### Frontend

- Log in with test credentials
- Verify token is stored in secure store
- Fetch tracks → should only show current user's tracks
- Play a track → stream URL should include token
- Log out → token should be cleared, redirect to login

---

## Implementation Checklist

- [x] Add `users` table to schema (`006_users.sql`)
- [x] Add `user_id` column to `tracks` table (`007_tracks_user_id.sql`, destructive — wipes existing rows)
- [x] Install `golang-jwt/jwt/v5` and `golang.org/x/crypto/bcrypt`
- [x] Implement token generation + verification functions (`internal/auth`)
- [x] Add `POST /api/auth/register` endpoint
- [x] Add `POST /api/auth/login` endpoint
- [x] Protect `GET /api/tracks` with token verification (`withAuth` middleware + `s.protected`)
- [x] Protect `GET /api/tracks/{id}/stream` (+ `/artwork`) with token verification + ownership check (scoped `GetTrack`)
- [x] Update `POST /api/tracks` to associate with logged-in user (`IngestFile` takes `userID`)
- [x] Install `expo-secure-store` on frontend (with a `localStorage` fallback for web)
- [x] Implement AuthContext + useAuth hook (`app/src/hooks/useAuth.tsx`)
- [x] Build login/register screens (`app/src/app/(auth)/`)
- [x] Update navigation to show login when no token (`<Redirect>` guards in the group layouts + `upload.tsx`)
- [x] Update track list to attach token to requests (`client.ts` module token + `Authorization` header)
- [x] Update audio player to construct stream URL with token (`streamUrl`/`artworkUrl` append `?token=`)
- [x] Test end-to-end (register → login → upload → stream; server unit + curl, app typecheck/lint/export)

---

## Known Edge Cases

1. **Token expires mid-stream:** Handle 401 response by prompting re-login
2. **User deleted their account:** Tracks become orphaned (implement cascade delete in schema)
3. **Multiple devices:** Same user logs in on phone + web — should they share the same token? (For MVP: yes, assume same device)
4. **Password reset:** Not in scope for MVP (manual process for now)

---

## Estimate

- **Backend:** 2–3 days (JWT setup, endpoints, DB changes)
- **Frontend:** 2–3 days (auth context, screens, token attachment)
- **Testing & debugging:** 1–2 days

**Total: 1 week** (can be done in parallel; start with backend while UI is simple)

---

## Next After Auth

Once Phase 5 is done, you'll have:
- Multi-user support
- Secure token-based access
- Each user's library isolated

Then move to **Phase 6 (Playlists & Search)** or **Phase 7 (Bulk Import)** depending on priority.
