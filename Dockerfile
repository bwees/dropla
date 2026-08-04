FROM alpine:3.20 AS runtime
ARG TARGETPLATFORM
RUN apk add --no-cache tini ca-certificates
WORKDIR /app

# Installed on PATH as `dropla` so subcommands work through docker exec, e.g.
#   echo 'new-password' | docker exec -i dropla dropla reset-password alice
COPY ${TARGETPLATFORM}/app /usr/local/bin/dropla
COPY web/build /app/web
RUN chmod +x /usr/local/bin/dropla

ENV PORT=3000
ENV WEB_STATIC_PATH=/app/web

# Without these the SQLite file and every uploaded blob land in the image's
# writable layer and are lost on redeploy. Declared as a volume so running
# without an explicit mount still survives a container replacement.
ENV DATABASE_DIR=/data/db
ENV UPLOAD_DIR=/data/uploads
VOLUME /data

EXPOSE 3000

ENTRYPOINT ["/sbin/tini", "--", "dropla"]
