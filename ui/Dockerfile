ARG NODE_VERSION=24.21.0
ARG NGINX_VERSION=1.30.6

FROM node:${NODE_VERSION}-alpine AS builder
WORKDIR /src
COPY package.json package-lock.json* ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi
COPY . .
RUN npm run build

FROM nginxinc/nginx-unprivileged:${NGINX_VERSION}-alpine
# Links the GHCR package to the repo
LABEL org.opencontainers.image.source=https://github.com/bartlettc22/image-inquisitor
ENV API_UPSTREAM=http://imginq:8080
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=builder /src/dist /usr/share/nginx/html
EXPOSE 8080
