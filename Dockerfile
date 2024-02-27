FROM golang:1.22 AS build
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod tidy
COPY . .
RUN go build -o cpa .

FROM golang:1.22-alpine
WORKDIR /app
COPY --from=build /build/cpa /app/cpa
EXPOSE 8080
EXPOSE 9090
CMD ["sh", "-c", "/app/cpa"]