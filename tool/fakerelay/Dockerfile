FROM golang:1.17-buster

WORKDIR /src
EXPOSE 5000
COPY . .
RUN go build -o fakerelay

ENTRYPOINT ["./fakerelay"]
