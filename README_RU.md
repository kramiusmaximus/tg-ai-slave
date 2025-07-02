<h1 align="center">
    <img src="img/logo.png" width="220" />
    <div>
    OpenRouter
    <br>
    Bot
    </div>
</h1>

<h4 align="center">
    <a href="README.md">English (🇺🇸)</a> | <strong>Русский (🇷🇺)</strong>
</h4>

Этот проект позволяет за несколько минут запустить своего Telegram бота для общения с бесплатными и платными моделями ИИ через [OpenRouter](https://openrouter.ai), или локальной LLM, например, через [LM Studio](https://lmstudio.ai).

> [!NOTE]
> Этот репозиторий является форком проекта [openrouter-gpt-telegram-bot](https://github.com/deinfinite/openrouter-gpt-telegram-bot), который реализует форматирование `Markdown` в ответах бота и оптимизирует процесс запуска в контейнере.

<details>
    <summary>Пример</summary>
    <img src="./img/example.png">
</details>

## Подготовка

- Зарегистрируйтесь в [OpenRouter](https://openrouter.ai) и получите [API ключ](https://openrouter.ai/settings/keys).

- Создайте своего Telegram бота, используя [@BotFather](https://telegram.me/BotFather) и получите его API токен.

- Получите свой Telegram id, используя [@getmyid_bot](https://t.me/getmyid_bot).

> [!TIP]
> При запуске бота вы сможете увидеть в логах идентификаторы других пользователей, которым вы также сможете предоставить доступ к боту в дальнейшем.

## Запуск

- Создать рабочий каталог бота:

```bash
mkdir openrouter-bot
cd openrouter-bot
```

- Создайте `.env` файл и заполните основные переменные:

```bash
# OpenRouter api key
API_KEY=
# Список бесплатных моделей: https://openrouter.ai/models?max_price=0
MODEL=deepseek/deepseek-r1:free
# Telegram api key
TELEGRAM_BOT_TOKEN=
# Ваш Telegram id
ADMIN_IDS=
# Список пользователей для доступа к боту, разделенных запятыми
ALLOWED_USER_IDS=
# Отключить гостевой доступ
GUEST_BUDGET=0
# Язык, используемый в ответах бота
LANG=RU
```

Список всех параметров приведен в файле [.env.example](https://github.com/Lifailon/openrouter-bot/blob/main/.env.example).

- Загрузите образ из [Docker Hub](https://hub.docker.com/r/lifailon/openrouter-bot) и запустите контейнер:

```bash
docker run -d --name OpenRouter-Bot \
    -v ./.env:/openrouter-bot/.env \
    --restart unless-stopped \
    lifailon/openrouter-bot:latest
```

Образ собран для платформ `amd64` и `aarch64` (Raspberry Pi) с использованием [docker buildx](https://github.com/docker/buildx).

## Сборка

```bash
git clone https://github.com/Lifailon/openrouter-bot
cd openrouter-bot
docker-compose up -d --build
```
