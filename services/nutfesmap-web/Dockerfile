# ---- dev ----
FROM node:20-alpine AS dev
WORKDIR /app

ENV NODE_ENV=development

# 依存だけ先に入れてキャッシュを効かせる
COPY package*.json ./
RUN npm install

# 開発はソースをCOPYしない（↓composeでマウントするため）
# CMDでは 0.0.0.0 バインド & ポーリング有効化
ENV CHOKIDAR_USEPOLLING=true
ENV WATCHPACK_POLLING=true

EXPOSE 3000
CMD ["npm", "run", "dev", "--", "-H", "0.0.0.0"]
