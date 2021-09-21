FROM golang:1.14

ENV GO111MODULE on
WORKDIR /go-std

RUN go get github.com/valyala/quicktemplate/qtc
RUN go get -u github.com/mailru/easyjson/...

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src ./

RUN go generate ./templates
RUN easyjson -pkg
RUN go build -ldflags="-s -w" -o app .

EXPOSE 8080

CMD ./app -db pgx
