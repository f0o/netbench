FROM golang:alpine AS builder

ARG VERSION
ARG COMMIT
ARG BUILD_DATE

WORKDIR /opt/netbench

ADD . /opt/netbench

RUN go build -o netbench -ldflags "-s -w -X main.version='${VERSION}' -X main.commit='${COMMIT}' -X main.date='${BUILD_DATE}'" cmd/.

FROM alpine AS final

COPY --from=builder /opt/netbench/netbench /netbench

ENV ENV=prod
ENTRYPOINT [ "/netbench" ]

CMD [ "--help" ]
