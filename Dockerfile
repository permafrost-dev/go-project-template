FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY ./ /app

RUN apk update && apk add --no-cache curl bash make build-base ca-certificates jq wget python3 py3-pip
RUN sh -c "$(curl -s --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
RUN task build-release -fv

# build the final image, keeping only the necessary files
FROM golang:1.25-alpine

LABEL org.opencontainers.image.source=https://github.com/vendor-name/project-name
LABEL org.opencontainers.image.description="{{project.description}}"

RUN apk update && apk add --no-cache curl bash ca-certificates jq wget

COPY --from=builder /app/dist /app

ENTRYPOINT ["/app/server"]
