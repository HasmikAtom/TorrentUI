# Stage 1: build the React frontend
FROM node:20-alpine AS frontend-build
WORKDIR /app
RUN corepack enable
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build
# output: /app/dist

# Stage 2: build the auth-service TS
FROM node:20-alpine AS service-build
WORKDIR /app
RUN corepack enable
COPY auth-service/package.json auth-service/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY auth-service/ ./
RUN pnpm build

# Stage 3: runtime
FROM node:20-alpine
WORKDIR /app
RUN corepack enable
COPY auth-service/package.json auth-service/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --prod
COPY --from=service-build /app/dist ./dist
COPY --from=frontend-build /app/dist ./public

EXPOSE 3000
CMD ["node", "dist/server.js"]
