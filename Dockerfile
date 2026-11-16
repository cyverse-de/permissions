FROM golang:1.27 AS build

WORKDIR /permissions
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/permissions ./cmd/permissions-server

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /out/permissions /bin/permissions

USER nonroot:nonroot

ENTRYPOINT ["permissions"]
CMD ["--help"]

EXPOSE 60000

ARG git_commit=unknown
ARG version="2.9.0"
ARG descriptive_version=unknown

LABEL org.cyverse.git-ref="$git_commit"
LABEL org.cyverse.version="$version"
LABEL org.cyverse.descriptive-version="$descriptive_version"
LABEL org.label-schema.vcs-ref="$git_commit"
LABEL org.label-schema.vcs-url="https://github.com/cyverse-de/permissions"
LABEL org.label-schema.version="$descriptive_version"
