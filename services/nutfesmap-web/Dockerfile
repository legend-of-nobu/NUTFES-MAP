# services/nutfesmap-web/Dockerfile

# --- dev 用（ホットリロード） ---
FROM node:20-alpine AS dev
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
EXPOSE 3000
CMD ["npm","run","dev","--","-H","0.0.0.0"]

# --- prod 用（ビルド→start） ---
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
# 環境変数（NEXT_PUBLIC_*）は build args で注入される想定
ARG NEXT_PUBLIC_API_BASE_URL
ENV NEXT_PUBLIC_API_BASE_URL=${NEXT_PUBLIC_API_BASE_URL}
COPY . .
RUN npm run build

FROM node:20-alpine AS prod
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app ./
EXPOSE 3000
CMD ["npm","run","start","--","-H","0.0.0.0"]
