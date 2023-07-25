FROM golang:alpine AS builder

WORKDIR /opt/netbench

ADD . /opt/netbench

RUN go build cmd/netbench.go

FROM alpine AS final

COPY --from=builder /opt/netbench/netbench /netbench

ENV ENV=prod
ENTRYPOINT [ "/netbench" ]

CMD [ "--help" ]
