FROM golang:1.26-alpine3.24 as builder

ADD . /code
# Run tests
RUN go env && cd /code && go test -buildvcs=false ./...
# Compile the binary
RUN go env && cd /code && go build -buildvcs=false -o /gitwright .

FROM alpine:3.24

RUN wget -qO /usr/local/bin/docker-compose "https://github.com/docker/compose/releases/latest/download/docker-compose-linux-$(uname -m)" \
    && chmod +x /usr/local/bin/docker-compose

RUN mkdir /data

COPY --from=builder /gitwright /bin/gitwright

ENTRYPOINT ["gitwright"]
