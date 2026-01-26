FROM golang:latest as go-build-stage

ENV GOPROXY https://goproxy.cn,direct

WORKDIR /go/src/app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest

RUN apk add --no-cache tzdata ca-certificates

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=go-build-stage /go/src/app/main .

CMD ["./main"]