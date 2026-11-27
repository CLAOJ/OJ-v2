# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN --mount=type=cache,target=/root/.npm npm ci

COPY . .

# Build argument for API URL (optional - defaults to window.location.origin/api at runtime)
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}

RUN --mount=type=cache,target=/app/.next/cache npm run build

# Production stage
FROM node:20-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install production dependencies only
RUN --mount=type=cache,target=/root/.npm npm ci --omit=dev

# Copy built application
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/next.config.ts ./
COPY cluster.mjs ./

EXPOSE 3000

ENV NODE_ENV=production
ENV PORT=3000
# Server-rendered dates (dayjs/Date use the process time zone) must match what
# visitors' browsers compute, or React discards the HTML on hydration (#418).
# The audience is in Vietnam; elsewhere React re-renders with local times.
ENV TZ=Asia/Ho_Chi_Minh
# Next.js workers per container (see cluster.mjs); one process uses ~1.5 cores.
ENV WEB_CONCURRENCY=4

CMD ["node", "cluster.mjs"]
