## Запуск

### Для docker Compose:
```
make up
````
или 
```
docker compose up -d --build
```

Для docker-compose:
```
make compose-up  
```
или
```
docker-compose up -d --build
```


### При необходимости создать .env файл (в Makefile это уже предусмотрено)
```
cp .env.example .env
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
