FROM --platform=$BUILDPLATFORM docker.io/golang:alpine AS build-service
ARG TARGETOS TARGETARCH
ENV GOMODCACHE=/root/.cache/go-build
WORKDIR /src
COPY --link go.* .
RUN --mount=type=cache,target=/root/.cache/go-build go mod download
COPY --link . .
RUN --mount=type=cache,target=/root/.cache/go-build GOOS=$TARGETOS GOARCH=$TARGETARCH go build -tags=release,nomsgpack,go_json -ldflags="-s -w" -o /service .

FROM scratch

ENV GIN_MODE=release

COPY --from=build-service /etc/ssl/cert.pem /etc/ssl/cert.pem
COPY --from=build-service /service /service
ENTRYPOINT ["/service"]
EXPOSE 8000


