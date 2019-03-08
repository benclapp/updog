FROM golang:1.12 as builder

WORKDIR $GOPATH/src/github.com/benclapp/updog
COPY . .

RUN go get -d -v
RUN go build -o /app/updog


FROM scratch

COPY --from=builder /app/updog /app/updog
COPY config/*.yaml /config/

ENTRYPOINT [ "/app/updog" ]