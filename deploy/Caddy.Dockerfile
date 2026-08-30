# syntax=docker/dockerfile:1

# Build the Expo web app to static files. EXPO_PUBLIC_API_URL is inlined
# by Expo at export time, so it must be the production origin.
FROM node:22 AS web
WORKDIR /app
COPY app/package.json app/package-lock.json ./
RUN npm ci
COPY app/ .
ARG EXPO_PUBLIC_API_URL
RUN EXPO_PUBLIC_API_URL=$EXPO_PUBLIC_API_URL npx expo export -p web --output-dir dist

# Caddy serves the static export and reverse-proxies /api/* (see Caddyfile).
FROM caddy:2-alpine
COPY --from=web /app/dist /srv/www
COPY deploy/Caddyfile /etc/caddy/Caddyfile
