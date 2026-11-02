# SynAgent: one image, several modes (web | worker | migrate | mailcheck).
#
# The build stage runs on the build platform and cross-compiles, so building the
# linux/arm64 image for the Graviton node from an amd64 laptop needs no emulation.
#
#   docker buildx build --platform linux/arm64 -t synagent:dev --load .

FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS go
ARG TARGETOS=linux
ARG TARGETARCH=arm64
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/synagent ./cmd/synagent

# distroless/static: CA certificates and tzdata, no shell, runs as nonroot.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go /out/synagent /synagent
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/synagent"]
