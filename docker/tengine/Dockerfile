# Build stage — Tengine 3.1.0 source'dan derleme
FROM alpine:3.19 AS builder

RUN apk add --no-cache \
    gcc musl-dev linux-headers \
    openssl-dev pcre-dev zlib-dev \
    make curl

WORKDIR /build
RUN curl -fSL https://github.com/alibaba/tengine/archive/refs/tags/3.1.0.tar.gz | tar xz

WORKDIR /build/tengine-3.1.0
RUN ./configure \
    --prefix=/etc/tengine \
    --sbin-path=/usr/sbin/tengine \
    --conf-path=/etc/tengine/nginx.conf \
    --error-log-path=/var/log/tengine/error.log \
    --http-log-path=/var/log/tengine/access.log \
    --pid-path=/var/run/tengine.pid \
    --lock-path=/var/run/tengine.lock \
    --with-http_ssl_module \
    --with-http_v2_module \
    --with-http_realip_module \
    --with-http_gzip_static_module \
    --with-http_stub_status_module \
    --with-stream \
    --with-stream_ssl_module \
    --add-module=modules/ngx_http_upstream_check_module \
    --add-module=modules/ngx_http_upstream_dynamic_module && \
    make -j$(nproc) && make install

# Runtime stage — minimal image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates openssl pcre zlib inotify-tools && \
    mkdir -p /var/log/tengine /etc/tengine/conf.d /var/cache/tengine /etc/tengine/html/.well-known/acme-challenge

COPY --from=builder /usr/sbin/tengine /usr/sbin/
COPY --from=builder /etc/tengine /etc/tengine

# Replace the default nginx.conf with our custom one (no default server block)
COPY nginx.conf /etc/tengine/nginx.conf
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 80 443

STOPSIGNAL SIGQUIT

ENTRYPOINT ["/entrypoint.sh"]
