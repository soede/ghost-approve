# Ghost-Approve Bot 👻

**Ghost-Approve Bot** — это проект на Go, который автоматизирует процессы согласования в бизнесе через чат-бота, интегрированного с VK Teams.

---

## 🛠 Как запустить проект

1. **Убедитесь, что у вас установлен Go** (минимальная версия Go 1.18)
2. **Скачайте зависимости:**
     ```bash
     go mod tidy 
   ``` 
3. **Настройте файл .env** <br>
   Создайте файл .env в корневой папке проекта:
    ```bash 
    touch .env
    ```
    **Заполните его следующим содержимым:** 
    ```plaintext
    DB_HOST=                 # Адрес базы данных
    DB_PORT=                 # Порт базы данных
    DB_USER=                 # Имя пользователя базы данных
    DB_PASSWORD=             # Пароль базы данных
    DB_NAME=                 # Имя базы данных
    SSL_MODE=                # Режим SSL
    VK_BOT_TOKEN=            # Токен бота VK
    REDIS_HOST=              # Адрес Redis
    REDIS_PORT=              # Порт Redis
    REDIS_PASSWORD=          # Пароль Redis 
    REDIS_DB=0               # Номер базы данных Redis
    ``` 
4. **Запустите проект**
    ```bash
    go run main.go
    ```
