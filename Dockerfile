FROM alpine:latest as alpine_builder
RUN apk --no-cache add ca-certificates
RUN apk --no-cache add tzdata

FROM scratch
COPY --from=alpine_builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=alpine_builder /usr/share/zoneinfo /usr/share/zoneinfo
ENTRYPOINT ["/dns-sync"]
COPY dns-sync /