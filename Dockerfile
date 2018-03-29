FROM gcr.io/gcp-runtimes/go1-builder:1.9 as build-env
ADD .  /usr/local/go/src/github.com/tarokamikaze/gae_task_resolver
RUN cd /usr/local/go/src/github.com/tarokamikaze/gae_task_resolver/ \
    && /usr/local/go/bin/go build server.go

# Application image.
FROM gcr.io/distroless/base:latest
COPY --from=build-env /usr/local/go/src/github.com/tarokamikaze/gae_task_resolver/server /usr/local/bin/server

CMD ["/usr/local/bin/server"]