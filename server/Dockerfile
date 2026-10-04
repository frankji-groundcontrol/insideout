FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/insideout ./cmd/insideout

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/insideout /insideout
# distroless/static ships a nonroot user (uid 65532) but defaults to root.
USER nonroot:nonroot
EXPOSE 8080
# No shell or curl in distroless: the binary probes its own /healthz
# (the `health` subcommand derives the port from INSIDEOUT_ADDR/PORT).
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD ["/insideout", "health"]
ENTRYPOINT ["/insideout"]
