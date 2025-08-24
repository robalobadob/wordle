FROM golang:1.22-alpine as builder
WORKDIR /app
COPY . .
RUN go build -o go-server .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/go-server /usr/local/bin/go-server
EXPOSE 5175
ENV PORT=5175
CMD ["go-server"]
