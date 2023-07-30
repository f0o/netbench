FROM golang:alpine AS builder

ARG VERSION="devel"
ARG COMMIT="0000000"
ARG BUILD_DATE="1970-01-01T00:00:00Z"

WORKDIR /opt/netbench

ADD . /opt/netbench

RUN cd cmd/netbench \
&& go build -o ../../netbench -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" . \
&& strip ../../netbench

FROM alpine AS final

COPY --from=builder /opt/netbench/netbench /netbench

ENV ENV=prod
ENTRYPOINT [ "/netbench" ]

CMD [ "--help" ]
