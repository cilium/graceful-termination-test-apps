FROM golang:1.16-alpine AS builder
WORKDIR /app
COPY server .
RUN go build -o server .
WORKDIR /app
COPY client .
RUN go build -o client .

FROM alpine:3.11
COPY --from=builder /app/server /server
COPY --from=builder /app/client /client
RUN chmod a+x /server
RUN chmod a+x /client
