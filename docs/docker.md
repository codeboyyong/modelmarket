# Docker Development and Multi-Architecture Builds

## Local Development

The local stack is defined in `docker-compose.yml`.

Start the app:

```sh
scripts/dev-up.sh
```

Reset local containers and data:

```sh
scripts/dev-reset.sh
```

The Compose helper supports both Docker Compose forms:

- `docker compose`
- `docker-compose`

If neither is installed, `scripts/compose.sh` will print a clear error.

## Docker on Mac

Recommended options:

- Docker Desktop: easiest install, includes Docker Compose and Buildx.
- Colima: lightweight Linux VM for Docker.
- Full Linux VM: heavier, closest to production Linux.

With Colima:

```sh
brew install colima docker docker-compose
colima start --cpu 4 --memory 8 --disk 40
docker version
docker compose version || docker-compose version
```

## Local Image Builds

Build images for the current machine architecture and load them into local Docker:

```sh
scripts/docker-build-local.sh
```

Build for a specific local architecture:

```sh
PLATFORM=linux/arm64 scripts/docker-build-local.sh
PLATFORM=linux/amd64 scripts/docker-build-local.sh
```

`--load` only supports one platform at a time.

## Multi-Architecture Registry Builds

Build images that run on both ARM and Intel/AMD Linux machines:

```sh
IMAGE_REGISTRY=ghcr.io \
IMAGE_NAMESPACE=your-org/model-market \
IMAGE_TAG=phase1 \
scripts/docker-build-multiarch.sh
```

By default this builds:

- `linux/amd64`
- `linux/arm64`

And pushes:

- `ghcr.io/your-org/model-market/backend:phase1`
- `ghcr.io/your-org/model-market/frontend:phase1`

You can override platforms:

```sh
PLATFORMS=linux/amd64,linux/arm64 scripts/docker-build-multiarch.sh
```

## Required Environment Variables

- `IMAGE_REGISTRY`: optional registry host, such as `ghcr.io`.
- `IMAGE_NAMESPACE`: image namespace, default `model-market`.
- `IMAGE_TAG`: image tag, default `phase1`.
- `PLATFORMS`: target platforms, default `linux/amd64,linux/arm64`.
- `OUTPUT_MODE`: `push` or `load`, default `push`.
- `BUILDER_NAME`: Buildx builder name, default `model-market-builder`.

## Notes

Multi-architecture builds usually require `OUTPUT_MODE=push` because Docker creates a manifest list in the registry. For local testing, build one platform with `OUTPUT_MODE=load`.

The current Dockerfiles use official multi-architecture base images:

- `golang:1.25-alpine`
- `node:24-alpine`
- `alpine:3.20`

Those support both `linux/amd64` and `linux/arm64`.
