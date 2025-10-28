APP_NAME := todo
BIN_DIR := bin
CMD_DIR := ./cmd/todo
FRONTEND_DIR := ./web
STATIC_DIR := $(FRONTEND_DIR)/dist
NPM := npm

.PHONY: all frontend frontend-payed backend clean run

all: frontend backend

frontend:
	@echo "==> Building of frontend skipped"

frontend-payed:
	@echo "==> Build frontend"
	cd $(FRONTEND_DIR) && [ -d node_modules ] || $(NPM) install
	cd $(FRONTEND_DIR) && $(NPM) run build

backend:
	@echo "==> Here you can add command for rsrc tool if you need icon for Windows EXE"
	@echo "==> Compile Go service"
	mkdir -p $(BIN_DIR)
	GO111MODULE=on go build -o $(BIN_DIR)/$(APP_NAME) $(CMD_DIR)

run: all
	./$(BIN_DIR)/$(APP_NAME)

clean:
	rm -rf $(BIN_DIR)
	rm -rf $(STATIC_DIR)
