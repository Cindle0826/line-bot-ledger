# Cloud Run only runs linux/amd64 images. Pinning --platform here means the
# image is always amd64 regardless of the host running `docker build` (e.g.
# an Apple Silicon Mac, which defaults to arm64) — the normal deploy path is
# `gcloud builds submit` (see deploy/webhook/cloudbuild.yaml), which already
# runs on amd64 Cloud Build workers, but this keeps a local `docker build`
# safe too.
FROM --platform=linux/amd64 golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/server ./services/webhook

FROM --platform=linux/amd64 gcr.io/distroless/static-debian12
COPY --from=build /out/server /server
ENTRYPOINT ["/server"]
