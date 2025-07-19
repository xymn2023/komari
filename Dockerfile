FROM debian:stable-slim

WORKDIR /app

# 安装依赖
RUN apt-get update && apt-get install -y ca-certificates gcc libc6-dev libsqlite3-0 && rm -rf /var/lib/apt/lists/*

COPY komari-linux-amd64 /app/komari

RUN chmod +x /app/komari

ENV GIN_MODE=release
ENV KOMARI_DB_TYPE=sqlite
ENV KOMARI_DB_FILE=/app/data/komari.db
ENV KOMARI_DB_HOST=localhost
ENV KOMARI_DB_PORT=3306
ENV KOMARI_DB_USER=root
ENV KOMARI_DB_PASS=
ENV KOMARI_DB_NAME=komari
ENV KOMARI_LISTEN=0.0.0.0:25774

EXPOSE 25774

CMD ["/app/komari", "server"]