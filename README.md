### Для запуска создать .env файл
```
cp .env.example .env
```

### Запустить 

```
docker-compose up 
```



## Доступ к базе данных (если нужно)

### Подключиться к БД внутри контейнера

```
docker compose exec db psql -U avito_user -d avito_db
```

# Или пробросить порт временно

```
docker compose exec -it db 
```

затем внутри:

```
psql -U avito_user -d avito_db
```
