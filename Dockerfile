FROM golang:alpine
WORKDIR /code
ENV TG_BOT_TOKEN="7602027535:AAGrPptEC__CbwGqT-6vFpc-SdnEBAAtiSY"
COPY . /code/
RUN go build -o api