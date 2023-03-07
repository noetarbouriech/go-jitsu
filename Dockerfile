# STAGE 1: build binary
FROM golang:1.20-alpine AS build-stage

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY game/ ./game
COPY server/ ./server
COPY *.go ./

RUN go build -o ./go-jitsu

# STAGE 2: run binary
FROM scratch AS run-stage

WORKDIR /app

COPY --from=build-stage /app/go-jitsu ./

#RUN apk add --no-cache openssh

EXPOSE 3000

CMD [ "./go-jitsu" ]
