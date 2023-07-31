FROM golang:1.19 AS builder
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go mod tidy
RUN ls
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest AS production
COPY --from=builder /app .
CMD ["./main"]