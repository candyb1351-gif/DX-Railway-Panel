# ========================================================
# Stage: Frontend (Vite)
# ========================================================
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm config set fetch-retries 5 \
  && npm config set fetch-retry-mintimeout 20000 \
  && npm config set fetch-retry-maxtimeout 120000 \
  && npm ci
COPY frontend/ ./
COPY internal/web/translation /src/internal/web/translation
RUN npm run build

# ========================================================
# Stage: Builder
# ========================================================
FROM golang:1.26-alpine AS builder
WORKDIR /app
ARG TARGETARCH

RUN apk --no-cache --update add \
  build-base \
  gcc \
  curl \
  unzip

COPY . .
COPY --from=frontend /src/internal/web/dist ./internal/web/dist

ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
RUN go build -ldflags "-w -s" -o build/DX main.go
RUN ./DockerInit.sh "$TARGETARCH"

# ========================================================
# Stage: Final Image of DX
# ========================================================
FROM alpine
ENV TZ=Asia/Tehran
WORKDIR /app

RUN apk add --no-cache --update \
  ca-certificates \
  tzdata \
  fail2ban \
  bash \
  curl \
  openssl

COPY --from=builder /app/build/ /app/
COPY --from=builder /app/DockerEntrypoint.sh /app/
COPY --from=builder /app/DX.sh /usr/bin/DX
COPY --from=builder /app/internal/web/translation /app/internal/web/translation
COPY railway-entrypoint.sh /app/


# Configure fail2ban
RUN rm -f /etc/fail2ban/jail.d/alpine-ssh.conf \
  && cp /etc/fail2ban/jail.conf /etc/fail2ban/jail.local \
  && sed -i "s/^\[ssh\]$/&\nenabled = false/" /etc/fail2ban/jail.local \
  && sed -i "s/^\[sshd\]$/&\nenabled = false/" /etc/fail2ban/jail.local \
  && sed -i "s/#allowipv6 = auto/allowipv6 = auto/g" /etc/fail2ban/fail2ban.conf

RUN chmod +x \
  /app/DockerEntrypoint.sh \
  /app/railway-entrypoint.sh \
  /app/DX \
  /usr/bin/DX

ENV DX_IN_DOCKER="true"
ENV DX_ENABLE_FAIL2BAN="true"
ENV DX_DB_TYPE=""
ENV DX_DB_DSN=""
EXPOSE 2053
# Note: no VOLUME instruction here -- Railway doesn't support Docker's VOLUME
# directive at build time. Persistence is handled by attaching a Railway
# Volume with mount path /etc/DX from the service's Settings > Volumes tab.
CMD [ "./DX" ]
# railway-entrypoint.sh syncs the panel port with Railway's $PORT (if present)
# then execs the normal DockerEntrypoint.sh -- on any other host $PORT is
# unset so behavior is 100% unchanged.
ENTRYPOINT [ "/app/railway-entrypoint.sh" ]
