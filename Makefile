# Use .env file for environment variables
include .env
export

# Default docker-compose file
COMPOSE_FILE := docker-compose.yml
# Detached mode flag
D_FLAG := -d

# Check if dev mode is specified
ifeq ($(DEV_MODE),true)
    COMPOSE_FILE := docker-compose.dev.yml
	D_FLAG :=
endif

up:
	docker-compose -f $(COMPOSE_FILE) up --build $(D_FLAG)

down:
	docker-compose -f $(COMPOSE_FILE) down