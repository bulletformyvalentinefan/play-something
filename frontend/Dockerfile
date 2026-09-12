FROM node:20-alpine AS base

RUN npm i -g pnpm@10.2.1

WORKDIR /app

# Build stage
FROM base AS build

# Copy dependency files first (cache layer)
COPY package.json pnpm-lock.yaml ./

# Install dependencies with pnpm
RUN --mount=type=cache,target=/root/.pnpm-store \
    pnpm install --include=dev

# Copy application code
COPY . .

# Build with Vite
RUN pnpm build

# Runtime stage: nginx with optimizations
FROM nginx:stable-alpine

# Add gzip compression and security headers
RUN rm /etc/nginx/conf.d/default.conf

COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:80 || exit 1

EXPOSE 80

# Run nginx
CMD ["nginx", "-g", "daemon off;"]
