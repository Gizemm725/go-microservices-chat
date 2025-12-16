# 1. Aşama: İnşaat (Build)
# Golang yüklü küçük bir Linux kullanıyoruz
FROM golang:1.25-alpine AS builder

# Çalışma klasörünü ayarla
WORKDIR /app

# Kütüphane listesini kopyala ve indir
COPY go.mod go.sum ./
RUN go mod download

# Tüm kodları kopyala
COPY . .

# Uygulamayı derle (build) ve 'main' adında bir dosya çıkar
RUN go build -o main .

# 2. Aşama: Çalıştırma (Run)
# Sadece derlenmiş dosyayı alacağımız tertemiz, küçücük bir Linux
FROM alpine:latest

WORKDIR /root/

# İlk aşamadan sadece 'main' dosyasını al
COPY --from=builder /app/main .

# Bu porttan yayın yapacak
EXPOSE 8080

# Başlat komutu
CMD ["./main"]