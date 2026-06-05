FROM node:24-alpine AS build

WORKDIR /app
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install
COPY frontend/ ./
RUN npm run build

FROM node:24-alpine

RUN adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=build /app/dist ./dist
COPY --from=build /app/server.mjs ./server.mjs
USER appuser

ENV PORT=3000
EXPOSE 3000
CMD ["node", "server.mjs"]
