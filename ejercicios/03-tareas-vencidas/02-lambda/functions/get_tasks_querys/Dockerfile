# syntax=docker/dockerfile:1.7
FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -mod=mod -trimpath -buildvcs=false -tags lambda.norpc -ldflags='-s -w' -o /out/bootstrap .

FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /out/bootstrap ${LAMBDA_RUNTIME_DIR}/bootstrap
CMD ["bootstrap"]
