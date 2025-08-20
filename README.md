<h1 align="center">
    <img src="img/logo.png" width="220" />
    <div>
    OpenRouter
    <br>
    Bot
    </div>
</h1>

<h4 align="center">
    <strong>English (🇺🇸)</strong> | <a href="README_RU.md">Русский (🇷🇺)</a>
</h4>

This project allows you to launch your Telegram bot in a few minutes to communicate with free and paid AI models via [OpenRouter](https://openrouter.ai), or local LLMs, for example, via [LM Studio](https://lmstudio.ai).

> [!NOTE]
> This repository is a fork of the [openrouter-gpt-telegram-bot](https://github.com/deinfinite/openrouter-gpt-telegram-bot) project, which adds new features (such as switch current model and `Markdown` formatting in bot responses) and optimizes the container startup process.

<details>
    <summary>Example</summary>
    <img src="./img/example.png">
    <img src="./img/commands.png">
</details>

## Preparation

- Register with [OpenRouter](https://openrouter.ai) and get an [API key](https://openrouter.ai/settings/keys).

- Create your Telegram bot using [@BotFather](https://telegram.me/BotFather) and get its API token.

- Get your telegram id using [@getmyid_bot](https://t.me/getmyid_bot).

> [!TIP]
> When you launch the bot, you will be able to see the IDs of other users in the log, to whom you can also grant access to the bot in the future.

## Installation

To run locally on Windows or Linux system, download the pre-built binary (without dependencies) from the [releases](https://github.com/Lifailon/openrouter-bot/releases) page.

### Running in Docker

- Create a working directory:

```bash
mkdir openrouter-bot
cd openrouter-bot
```

- Create `.env` file and fill in the basic variables:

```bash
# OpenRouter api key
API_KEY=
# Free modeles: https://openrouter.ai/models?max_price=0
MODEL=deepseek/deepseek-r1:free
# Telegram api key
TELEGRAM_BOT_TOKEN=
# Your Telegram id
ADMIN_IDS=
# List of users to access the bot, separated by commas
ALLOWED_USER_IDS=
# Disable guest access
GUEST_BUDGET=0
# Language used for bot responses (supported: EN/RU)
LANG=EN
```

The list of all parameters is listed in the [.env.example](https://github.com/Lifailon/openrouter-bot/blob/main/.env.example) file

- Run a container using the image from [Docker Hub](https://hub.docker.com/r/lifailon/openrouter-bot):

```bash
docker run -d --name OpenRouter-Bot \
    -v ./.env:/openrouter-bot/.env \
    --restart unless-stopped \
    lifailon/openrouter-bot:latest
```

The image is build for `amd64` and `arm64` (Raspberry Pi) platforms using [docker buildx](https://github.com/docker/buildx).

## Build

```bash
git clone https://github.com/Lifailon/openrouter-bot
cd openrouter-bot
docker-compose up -d --build
```
