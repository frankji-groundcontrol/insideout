FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/insideout ./cmd/insideout

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/insideout /insideout
EXPOSE 8080
ENTRYPOINT ["/insideout"]
