FROM golang:1.6-alpine

RUN mkdir -p /usr/src/app
WORKDIR /usr/src/app
COPY . /usr/src/app

RUN go build

EXPOSE 6789

CMD ./canaria-api
