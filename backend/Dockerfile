FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o main .
COPY wait-for-db.sh .
RUN chmod +x wait-for-db.sh
CMD ["./wait-for-db.sh", "./main"]
