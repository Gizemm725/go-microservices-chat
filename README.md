# 🚀  Dağıtık Sohbet & Analiz Platformu

Bu proje, Go (Fiber), React, PostgreSQL, MQTT, Kafka ve ClickHouse gibi modern teknolojileri bir araya getiren tam teşekküllü, uçtan uca Microservices (Mikro Servisler) mimarisinin bir uygulamasıdır.

## 🌟 Proje Mimarisi

Sistem, olay güdümlü (event-driven) bir yaklaşımla tasarlanmıştır. Kullanıcıdan gelen her mesaj, anında iletilmekle kalmaz, aynı zamanda analiz için Kafka üzerinden ClickHouse'a taşınır.



## 🛠️ Kullanılan Teknolojiler

| Katman | Servis | Teknoloji | Amaç |
| :--- | :--- | :--- | :--- |
| **Ön Yüz (Frontend)** | `frontend` | React, Nginx | Kullanıcı arayüzü ve oturum yönetimi. |
| **API/Backend** | `user-service` | Go (Fiber) | Kullanıcı kaydı ve JWT ile kimlik doğrulama. |
| **Chat Logic** | `chat-service` | Go (Fiber) | MQTT ile anlık mesajlaşma, Kafka'ya veri gönderme. |
| **Analiz İşçisi** | `metrics-service` | Go | Kafka'dan gelen mesajları okur ve ClickHouse'a işler. |
| **Veri Borusu** | `kafka` | Apache Kafka | Olayları (mesajları) gerçek zamanlı taşıyan yük kamyonu. |
| **Analiz DB** | `clickhouse` | ClickHouse | Yüksek performanslı analitik sorgular için sütun tabanlı veritabanı. |
| **İlişkisel DB** | `postgres-db` | PostgreSQL | Kullanıcılar ve sohbet geçmişi gibi kritik verileri saklar. |
| **Anlık İletim** | `emqx` | MQTT Broker | Cihazlar arası düşük gecikmeli mesaj iletimi. |

## ⚙️ Kurulum ve Çalıştırma

Projenin tamamı Docker konteynerleri üzerinde çalışacak şekilde yapılandırılmıştır. Tüm sistemi tek bir komutla ayağa kaldırabilirsiniz.

### Ön Koşullar

* **Docker** ve **Docker Compose** kurulu olmalıdır.

### Başlatma

Ana dizinde (`twinup-project`) terminali açın ve komutu çalıştırın:

```bash
docker-compose -f docker/docker-compose.yml up -d --build
