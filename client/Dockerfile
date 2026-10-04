# Pin to 3.44.0 (Dart 3.12.0). The floating :stable tag on GHCR lagged
# the local Flutter 3.44.8 / Dart 3.12.2 toolchain.
FROM ghcr.io/cirruslabs/flutter:3.44.0 AS build
WORKDIR /src
COPY pubspec.yaml pubspec.lock ./
RUN flutter pub get
COPY . .
# Same-origin /api/v1 via nginx; include any /v1 on the Go side already.
ARG API_BASE=/api/v1
RUN flutter build web --release --dart-define=API_BASE=${API_BASE}

# Unprivileged variant: nginx runs as uid 101 with no root master process,
# listening on 8080 — the same container port as the old nginx:1.27-alpine
# setup, so port mappings stay unchanged.
FROM nginxinc/nginx-unprivileged:1.27-alpine
# The image entrypoint envsubst's /etc/nginx/templates/*.template at
# container start; the baked NGINX_API_UPSTREAM default keeps Railway
# working, and docker-compose overrides it with the `server` service.
ENV NGINX_API_UPSTREAM=http://server.railway.internal:8080
COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=build /src/build/web /usr/share/nginx/html
USER 101
EXPOSE 8080
# busybox wget hits the /healthz stub (nginx.conf); no shell needed.
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD ["wget", "-qO-", "http://127.0.0.1:8080/healthz"]
CMD ["nginx", "-g", "daemon off;"]
