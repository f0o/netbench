FROM golang:alpine AS builder

ADD . /opt

WORKDIR /opt

RUN go build cmd/netbench.go

FROM alpine AS final

COPY --from=builder /opt/netbench /netbench

ENTRYPOINT [ "/netbench" ]

CMD [ "--help" ]
