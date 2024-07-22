FROM golang:alpine as build-env
COPY . /src
WORKDIR /src
RUN go build -o gowon-trakt

FROM alpine:3.20.2
RUN mkdir /data
ENV GOWON_TRAKT_KV_PATH /data/kv.db
WORKDIR /app
COPY --from=build-env /src/gowon-trakt /app/
ENTRYPOINT ["./gowon-trakt"]
