FROM golang:1.17-buster

WORKDIR /src

EXPOSE 5000

COPY go.mod go.sum* ./
RUN go mod download

COPY . ./
RUN go build -o fakerelay

ENTRYPOINT ["./fakerelay"]
