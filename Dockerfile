FROM golang:1.6-alpine

ENV GOPATH /usr/go
ENV SRCPATH $GOPATH/src/github.com/danielkrainas/canaria-api
RUN mkdir -p $SRCPATH

RUN mkdir -p $SRCPATH
WORKDIR $SRCPATH
COPY . $SRCPATH

RUN go build

EXPOSE 6789

CMD ./canaria-api
