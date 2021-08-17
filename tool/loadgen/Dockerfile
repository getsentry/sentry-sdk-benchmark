FROM golang:1.17-buster

WORKDIR /src
COPY . .
RUN go build -o loadgen

ENTRYPOINT ["./loadgen"]
