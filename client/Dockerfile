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

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/build/web /usr/share/nginx/html
EXPOSE 8080
# Railway healthcheck hits / or /healthz.
CMD ["nginx", "-g", "daemon off;"]
