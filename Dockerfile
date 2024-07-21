FROM golang:1.21-alpine AS build

WORKDIR /usr/src/up-bank-exporter

# Cache go modules during development
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go generate ./... && \
    go build -v -o /usr/local/bin/up-bank-exporter .

FROM alpine AS run

COPY --from=build /usr/local/bin/up-bank-exporter /usr/local/bin/up-bank-exporter

CMD ["up-bank-exporter"]
