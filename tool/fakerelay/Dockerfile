FROM golang:1.17-alpine as build-env

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /go/bin/fakerelay

FROM scratch

EXPOSE 5000

COPY --from=build-env /go/bin/fakerelay /fakerelay

ENTRYPOINT ["/fakerelay"]
