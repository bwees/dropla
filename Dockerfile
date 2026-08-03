FROM alpine:3.20 AS runtime
ARG TARGETPLATFORM
RUN apk add --no-cache tini ca-certificates
WORKDIR /app

COPY ${TARGETPLATFORM}/app /app/server/app
COPY web/build /app/web
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh /app/server/app

ENV PORT=3000
ENV WEB_STATIC_PATH=/app/web

EXPOSE 3000

WORKDIR /app/server
ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
