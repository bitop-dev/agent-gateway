FROM alpine:3.21

ARG TARGETARCH

RUN apk add --no-cache ca-certificates tzdata

COPY dist/${TARGETARCH}/gateway-linux-${TARGETARCH} /usr/local/bin/gateway

RUN chmod +x /usr/local/bin/gateway

EXPOSE 8080

ENTRYPOINT ["gateway"]
CMD ["--addr", ":8080"]
