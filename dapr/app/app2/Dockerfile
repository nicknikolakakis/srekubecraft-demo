# --- Builder stage: install deps + compile TypeScript ---
FROM node:22-alpine AS builder

WORKDIR /build

COPY package.json package-lock.json* ./
RUN npm ci --ignore-scripts

COPY tsconfig.json ./
COPY src/ ./src/
RUN npx tsc

# --- Runtime stage: production dependencies only ---
FROM node:22-alpine

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci --omit=dev --ignore-scripts && npm cache clean --force

COPY --from=builder /build/dist/ ./dist/
COPY public/ ./public/

RUN adduser -D -u 10001 appuser
USER 10001

EXPOSE 8080

CMD ["node", "dist/server.js"]
