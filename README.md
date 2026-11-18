# Описание
CMDB харнит информацию о серверах и виртуальных машинах:
- Имя сервера (hostname)
- IP-адрес
- Операционная система
- Окружение (dev, stage, prod)
- Статус (running, stopped, maintenance)
- Количество CPU и RAM
- Ответственная команда

## Запросы 

```bash
curl http://localhost:8080/health

# получить все серверы
curl http://localhost:8080/api/servers

# получить только prod серверы
curl "http://localhost:8080/api/servers?env=prod"

# добавить сервер
curl -X POST http://localhost:8080/api/servers \
  -H "Content-Type: application/json" \
  -d '{
    "hostname": "monitoring-01",
    "ip_address": "192.168.10.50",
    "os": "Debian 12",
    "environment": "prod",
    "status": "running",
    "cpu": 2,
    "ram": 4096,
    "team": "devops"
  }'

# получить сервер по id
curl http://localhost:8080/api/servers/1

# удалить сервер
curl -X DELETE http://localhost:8080/api/servers/6
```
