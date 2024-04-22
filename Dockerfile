# Строим бинарный файл
FROM golang:1.22 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/binary1
RUN CGO_ENABLED=0 GOOS=linux go build -o main2 ./cmd/binary2  # Компилируем binary2

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/main2 .  


EXPOSE 8080  

CMD sh -c './main & ./main2 & wait'
